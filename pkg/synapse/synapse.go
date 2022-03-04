package synapse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/utils"
	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/backoff"
	"github.com/spf13/viper"
)

// All constant related to synapse
const (
	Repo                   = "repo"
	BuildID                = "build-id"
	JobID                  = "job-id"
	Mode                   = "mode"
	ID                     = "id"
	DuplicateConnectionErr = "Duplicate connection"
	AuthenticationFailed   = "Authentication failed"
)

type synapse struct {
	conn                     *websocket.Conn
	runner                   core.DockerRunner
	secretsManager           core.SecretsManager
	logger                   lumber.Logger
	MsgErrChan               chan struct{}
	MsgChan                  chan []byte
	ConnectionAborted        chan struct{}
	InvalidConnectionRequest chan struct{}
	LogoutRequired           bool
}

// New returns new instance of synapse
func New(
	runner core.DockerRunner,
	logger lumber.Logger,
	secretsManager core.SecretsManager,
) core.SynapseManager {

	return &synapse{
		runner:                   runner,
		logger:                   logger,
		secretsManager:           secretsManager,
		MsgErrChan:               make(chan struct{}),
		InvalidConnectionRequest: make(chan struct{}),
		MsgChan:                  make(chan []byte, 1024),
		ConnectionAborted:        make(chan struct{}, 10),
		LogoutRequired:           true,
	}
}

func (s *synapse) InitiateConnection(
	ctx context.Context,
	wg *sync.WaitGroup,
	connectionFailed chan struct{}) {
	defer wg.Done()
	go s.openAndMaintainConnection(ctx, connectionFailed)
	<-ctx.Done()
	if s.LogoutRequired {
		s.logout()
	}
	s.runner.KillRunningDocker(context.TODO())
	s.logger.Debugf("exiting synapse")
}

func (s *synapse) openAndMaintainConnection(ctx context.Context, connectionFailed chan struct{}) {
	// setup exponential backoff for retrying control websocket connection
	var policy = backoff.NewExponential(
		backoff.WithInterval(500*time.Millisecond),           // base interval
		backoff.WithJitterFactor(0.05),                       // 5% jitter
		backoff.WithMaxRetries(global.MaxConnectionAttempts), // If not specified, default number of retries is 10
	)

	normalCloser := make(chan struct{})

	b, cancel := policy.Start(context.Background())
	defer cancel()
	s.logger.Debugf("starting socket connection")
	s.logger.Errorf("starting socket connection at URL %s", global.SocketURL[viper.GetString("env")])
	for backoff.Continue(b) {
		s.logger.Debugf("trying to connect to lamdatest server")
		select {
		case <-ctx.Done():
			return
		default:
			conn, _, err := websocket.DefaultDialer.Dial(global.SocketURL[viper.GetString("env")], nil)
			if err != nil {
				s.logger.Errorf("error connecting synapse to lambdatest %+v", err)
				continue
			}
			s.conn = conn
			s.logger.Debugf("synapse connected to lambdatest server")
			s.login()
			go s.handleIncomingMessage(normalCloser)
			go s.WriteMessage()
			select {
			case <-ctx.Done():
				s.ConnectionAborted <- struct{}{}
				return
			case <-normalCloser:
				conn.Close()
				s.ConnectionAborted <- struct{}{}
				return
			case <-s.InvalidConnectionRequest:
				conn.Close()
				s.ConnectionAborted <- struct{}{}
				connectionFailed <- struct{}{}
				s.LogoutRequired = false
				return
			case <-s.MsgErrChan:
				s.logger.Errorf("Connection between synpase and lambdatest break")
				s.ConnectionAborted <- struct{}{}
				conn.Close()
			}
			s.MsgErrChan = make(chan struct{})
			go s.openAndMaintainConnection(ctx, connectionFailed)
			return

		}
	}
	s.logger.Errorf("Unable to establish connection with lambdatest server. exiting...")
	connectionFailed <- struct{}{}
	s.LogoutRequired = false
}

func (s *synapse) handleIncomingMessage(normalCloser chan struct{}) {

	// s.conn.SetReadLimit(maxMessageSize)
	// s.conn.SetReadDeadline(time.Now().Add(pingWait))
	s.conn.SetPingHandler(func(string) error {
		if err := s.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
			s.logger.Errorf("Error in writing pong msg %s", err.Error())
			s.MsgErrChan <- struct{}{}
			close(s.MsgErrChan)
			return err
		}
		return nil
	})

	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				s.logger.Debugf("Normal closure occurred...........")
				normalCloser <- struct{}{}
				return
			}
			s.logger.Errorf("disconnecting from lambdatest server. error in reading message %v", err)
			s.MsgErrChan <- struct{}{}
			close(s.MsgErrChan)
			return
		}
		s.processMessage(msg)
	}
}

func (s *synapse) processMessage(msg []byte) {
	var message core.Message
	err := json.Unmarshal(msg, &message)
	if err != nil {
		s.logger.Errorf("error unmarshaling message")
	}

	switch message.Type {
	case core.MsgError:
		s.logger.Debugf("error message received from server")
		go s.processErrorMessage(message)
	case core.MsgInfo:
		s.logger.Debugf("info message received from server")
	case core.MsgTask:
		s.logger.Debugf("task message received from server")
		go s.processTask(message)
	default:
		s.logger.Errorf("message type not found")
	}
}

func (s *synapse) processErrorMessage(message core.Message) {
	errMsg := string(message.Content)
	s.logger.Errorf("error message received from server, error %s ", errMsg)
	if errMsg == DuplicateConnectionErr || errMsg == AuthenticationFailed {
		s.InvalidConnectionRequest <- struct{}{}
	}

}

func (s *synapse) processTask(message core.Message) {
	var runnerOpts core.RunnerOptions
	err := json.Unmarshal(message.Content, &runnerOpts)
	if err != nil {
		s.logger.Errorf("error unmarshaling core.task")
	}

	// sending job started updates
	if runnerOpts.PodType == core.NucleusPod {
		jobInfo := CreateJobInfo(core.JobStarted, &runnerOpts)
		s.logger.Infof("Sending update to neuron %+v", jobInfo)
		resourceStatsMessage := CreateJobUpdateMessage(jobInfo)
		s.SendMessage(&resourceStatsMessage)
	}
	// mounting secrets to container
	runnerOpts.HostVolumePath = fmt.Sprintf("/tmp/synapse/data/%s", runnerOpts.ContainerName)

	if err := utils.CreateDirectory(runnerOpts.HostVolumePath); err != nil {
		s.logger.Errorf("error creating file directory: %v", err)
	}
	if err := s.secretsManager.WriteGitSecrets(runnerOpts.HostVolumePath); err != nil {
		s.logger.Errorf("error creating secrets %v", err)
	}

	if err := s.secretsManager.WriteRepoSecrets(runnerOpts.Label[Repo], runnerOpts.HostVolumePath); err != nil {
		s.logger.Errorf("error creating repo secrets %v", err)
	}
	s.runAndUpdateJobStatus(runnerOpts)

}

func (s *synapse) runAndUpdateJobStatus(runnerOpts core.RunnerOptions) {
	// starting container
	statusChan := make(chan core.ContainerStatus)
	defer close(statusChan)
	s.logger.Debugf("starting container %s for build %s...", runnerOpts.ContainerName, runnerOpts.Label[BuildID])
	go s.runner.Initiate(context.TODO(), &runnerOpts, statusChan)

	status := <-statusChan
	// post job completion steps
	s.logger.Debugf("Status %+v", status)

	s.sendResourceUpdates(core.ResourceRelease, runnerOpts)
	jobStatus := core.JobFailed
	if status.Done {
		jobStatus = core.JobCompleted
	}
	jobInfo := CreateJobInfo(jobStatus, &runnerOpts)
	s.logger.Infof("Sending update to neuron %+v", jobInfo)
	resourceStatsMessage := CreateJobUpdateMessage(jobInfo)
	s.SendMessage(&resourceStatsMessage)
}

func (s *synapse) login() {
	cpu, ram := s.runner.GetInfo(context.TODO())
	id, err := machineid.ProtectedID("synapaseMeta")
	if err != nil {
		s.logger.Fatalf("Error while generating unique id")
	}
	lambdatestConfig := s.secretsManager.GetLambdatestSecrets()
	loginDetails := core.LoginDetails{
		SecretKey: lambdatestConfig.SecretKey,
		CPU:       cpu,
		RAM:       ram,
		SynapseID: id,
	}
	s.logger.Infof("Login synapse with id %s", loginDetails.SynapseID)

	loginMessage := CreateLoginMessage(loginDetails)
	s.SendMessage(&loginMessage)
}

func (s *synapse) logout() {
	s.logger.Infof("Logging out from lambdatest server")
	logoutMessage := CreateLogoutMessage()
	messageJson, err := json.Marshal(logoutMessage)

	if err != nil {
		s.logger.Errorf("error marshaling message")
		return
	}
	if err := s.conn.WriteMessage(websocket.TextMessage, messageJson); err != nil {
		s.logger.Errorf("error sending message to the server, error %v", err)

	}
}

func (s *synapse) sendResourceUpdates(
	status core.StatType,
	runnerOpts core.RunnerOptions,
) {
	specs := GetResources(runnerOpts.Tier)
	resourceStats := core.ResourceStats{
		Status: status,
		CPU:    specs.CPU,
		RAM:    specs.RAM,
	}
	resourceStatsMessage := CreateResourceStatsMessage(resourceStats)
	s.SendMessage(&resourceStatsMessage)
}

func (s *synapse) SendMessage(message *core.Message) {
	messageJson, err := json.Marshal(message)
	if err != nil {
		s.logger.Errorf("error marshaling message")
		return
	}
	s.MsgChan <- messageJson
}

func (s *synapse) WriteMessage() {
	for {
		select {
		case <-s.ConnectionAborted:
			return
		case messageJson := <-s.MsgChan:
			if err := s.conn.WriteMessage(websocket.TextMessage, messageJson); err != nil {
				s.logger.Errorf("error sending message to the server error %v", err)
				s.MsgChan <- messageJson
				s.MsgErrChan <- struct{}{}
				close(s.MsgErrChan)
				return
			}
		}
	}
}
