package synapse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/tasconfigdownloader"
	"github.com/cenkalti/backoff/v4"
	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

// All constant related to synapse
const (
	Repo                             = "repo"
	BuildID                          = "build-id"
	JobID                            = "job-id"
	Mode                             = "mode"
	ID                               = "id"
	DuplicateConnectionErr           = "Synapse already has an open connection"
	AuthenticationFailed             = "Synapse authentication failed"
	duplicateConnectionSleepDuration = 15 * time.Second
)

var buildAbortMap = make(map[string]bool)

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
	tasConfigDownloader      *tasconfigdownloader.TASConfigDownloader
}

// New returns new instance of synapse
func New(
	runner core.DockerRunner,
	logger lumber.Logger,
	secretsManager core.SecretsManager,
	tasConfigDownloader *tasconfigdownloader.TASConfigDownloader,
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
		tasConfigDownloader:      tasConfigDownloader,
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

/*
openAndMaintainConnection tries to create and mantain connection with
exponential backoff factor
*/
func (s *synapse) openAndMaintainConnection(ctx context.Context, connectionFailed chan struct{}) {
	// setup exponential backoff for retrying control websocket connection
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = 500 * time.Millisecond
	exponentialBackoff.RandomizationFactor = 0.05
	exponentialBackoff.MaxElapsedTime = 10 * time.Minute
	s.logger.Debugf("starting socket connection at URL %s", global.SocketURL[viper.GetString("env")])
	operation := func() error {
		s.logger.Debugf("trying to connect to TAS server")
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, _, err := websocket.DefaultDialer.Dial(global.SocketURL[viper.GetString("env")], nil)
			if err != nil {
				s.logger.Errorf("error connecting synapse to TAS %+v", err)
				return err
			}
			s.conn = conn
			s.logger.Debugf("synapse connected to TAS server")
			s.login()
			if !s.connectionHandler(ctx, conn, connectionFailed) {
				return nil
			}
			s.MsgErrChan = make(chan struct{})
			// re-listen for any connection breaks
			go s.openAndMaintainConnection(ctx, connectionFailed)
			return nil
		}
	}
	if err := backoff.Retry(operation, exponentialBackoff); err != nil {
		s.logger.Errorf("Unable to establish connection with lambdatest server. exiting...")
		connectionFailed <- struct{}{}
		s.LogoutRequired = false
	}
}

/*
 connectionHandler handles the connection by listening to any connection closer
 also it returns boolean value which represents whether we can retry to connect
*/
func (s *synapse) connectionHandler(ctx context.Context, conn *websocket.Conn, connectionFailed chan struct{}) bool {
	normalCloser := make(chan struct{})
	ctxDone := false
	defer func() {
		// if gracefully terminated, wait for logout message to be sent
		if !ctxDone {
			conn.Close()
		}
		s.ConnectionAborted <- struct{}{}
	}()

	go s.messageReader(normalCloser, conn)
	go s.messageWriter(conn)
	select {
	case <-ctx.Done():
		ctxDone = true
		return false
	case <-normalCloser:
		return false
	case <-s.InvalidConnectionRequest:
		connectionFailed <- struct{}{}
		s.LogoutRequired = false
		return false
	case <-s.MsgErrChan:
		s.logger.Errorf("Connection between synpase and lambdatest break")
		return true
	}
}

/*
messageReader reads websocket messages and acts upon it
*/
func (s *synapse) messageReader(normalCloser chan struct{}, conn *websocket.Conn) {
	conn.SetReadLimit(global.MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(global.PingWait)); err != nil {
		s.logger.Errorf("Error in setting read deadline , error: %v", err)
		s.MsgErrChan <- struct{}{}
		close(s.MsgErrChan)
		return
	}
	conn.SetPingHandler(func(string) error {
		if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
			s.logger.Errorf("Error in writing pong msg , error: %v", err)
			return err
		}
		if err := conn.SetReadDeadline(time.Now().Add(global.PingWait)); err != nil {
			s.logger.Errorf("Error in setting read deadline , error: %v", err)

			return err
		}
		return nil
	})
	duplicateConnectionChan := make(chan struct{})
	for {
		select {
		case <-duplicateConnectionChan:
			s.logger.Errorf("Duplicate connection detected .. will retry after certain time")
			time.Sleep(duplicateConnectionSleepDuration)
			s.MsgErrChan <- struct{}{}
			close(s.MsgErrChan)
			close(duplicateConnectionChan)
			return
		default:
			_, msg, err := conn.ReadMessage()
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
			s.processMessage(msg, duplicateConnectionChan)
		}
	}
}

// processMessage process messages received via websocket
func (s *synapse) processMessage(msg []byte, duplicateConnectionChan chan struct{}) {
	var message core.Message
	err := json.Unmarshal(msg, &message)
	if err != nil {
		s.logger.Errorf("error unmarshaling message")
	}

	switch message.Type {
	case core.MsgError:
		s.logger.Debugf("error message received from server")
		go s.processErrorMessage(message, duplicateConnectionChan)
	case core.MsgInfo:
		s.logger.Debugf("info message received from server")
	case core.MsgTask:
		s.logger.Debugf("task message received from server")
		go s.processTask(message)
	case core.MsgYMLParsingRequest:
		s.logger.Debugf("yml parsing request received from server")
		go s.processYMLParsingRequest(message)
	case core.MsgBuildAbort:
		s.logger.Debugf("abort-build message received from server")
		go s.processAbortBuild(message)
	default:
		s.logger.Errorf("message type not found")
	}
}

// processErrorMessage handles error messages
func (s *synapse) processErrorMessage(message core.Message, duplicateConnectionChan chan struct{}) {
	errMsg := string(message.Content)
	s.logger.Errorf("error message received from server, error %s ", errMsg)
	if errMsg == AuthenticationFailed {
		s.InvalidConnectionRequest <- struct{}{}
	}
	if errMsg == DuplicateConnectionErr {
		duplicateConnectionChan <- struct{}{}
	}
}

// processAbortBuild handles aborting a running build
func (s *synapse) processAbortBuild(message core.Message) {
	buildID := string(message.Content)
	buildAbortMap[buildID] = true
	s.logger.Debugf("message received to abort build %s", buildID)
	if err := s.runner.KillContainerForBuildID(buildID); err != nil {
		s.logger.Errorf("error while terminating container for buildID: %s, error: %v", buildID, err)
		return
	}
}

// processTask handles task type message
func (s *synapse) processTask(message core.Message) {
	var runnerOpts core.RunnerOptions
	err := json.Unmarshal(message.Content, &runnerOpts)
	if err != nil {
		s.logger.Errorf("error unmarshaling core.task")
	}

	// sending job started updates
	if runnerOpts.PodType == core.NucleusPod {
		jobInfo := CreateJobInfo(core.JobStarted, &runnerOpts, "")
		s.logger.Infof("Sending update to neuron %+v", jobInfo)
		resourceStatsMessage := CreateJobUpdateMessage(jobInfo)
		s.writeMessageToBuffer(&resourceStatsMessage)
	}
	// mounting secrets to container
	runnerOpts.HostVolumePath = fmt.Sprintf("/tmp/synapse/data/%s", runnerOpts.ContainerName)

	s.runAndUpdateJobStatus(&runnerOpts)
}

// runAndUpdateJobStatus intiate and sends jobs status
func (s *synapse) runAndUpdateJobStatus(runnerOpts *core.RunnerOptions) {
	// starting container
	statusChan := make(chan core.ContainerStatus)
	defer close(statusChan)
	s.logger.Debugf("starting container %s for build %s...", runnerOpts.ContainerName, runnerOpts.Label[BuildID])
	go s.runner.Initiate(context.TODO(), runnerOpts, statusChan)

	status := <-statusChan
	// post job completion steps
	s.logger.Debugf("jobID %s, buildID %s  status  %+v", runnerOpts.Label[JobID], runnerOpts.Label[BuildID], status)

	s.sendResourceUpdates(core.ResourceRelease, runnerOpts, runnerOpts.Label[JobID], runnerOpts.Label[BuildID])
	jobStatus := core.JobFailed
	if status.Done {
		jobStatus = core.JobCompleted
	}
	if buildAbortMap[runnerOpts.Label[BuildID]] {
		jobStatus = core.JobAborted
	}
	jobInfo := CreateJobInfo(jobStatus, runnerOpts, status.Error.Message)
	s.logger.Infof("Sending update to neuron %+v", jobInfo)
	resourceStatsMessage := CreateJobUpdateMessage(jobInfo)
	s.writeMessageToBuffer(&resourceStatsMessage)
}

// login write login message to lambdatest server
func (s *synapse) login() {
	cpu, ram := s.runner.GetInfo(context.TODO())
	id, err := machineid.ProtectedID("synapaseMeta")
	if err != nil {
		s.logger.Fatalf("Error while generating unique id")
	}
	lambdatestConfig := s.secretsManager.GetLambdatestSecrets()
	loginDetails := core.LoginDetails{
		Name:           s.secretsManager.GetSynapseName(),
		SecretKey:      lambdatestConfig.SecretKey,
		CPU:            cpu,
		RAM:            ram,
		SynapseID:      id,
		SynapseVersion: global.SynapseBinaryVersion,
	}
	s.logger.Infof("Login synapse with id %s", loginDetails.SynapseID)

	loginMessage := CreateLoginMessage(loginDetails)
	s.writeMessageToBuffer(&loginMessage)
}

// logout writes logout message to lambdatest server
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

// sendResourceUpdates sends resource status of synapse
func (s *synapse) sendResourceUpdates(
	status core.StatType,
	runnerOpts *core.RunnerOptions,
	jobID, buildID string,
) {
	specs := GetResources(runnerOpts.Tier)
	resourceStats := core.ResourceStats{
		Status: status,
		CPU:    specs.CPU,
		RAM:    specs.RAM,
	}
	s.logger.Debugf("sending resource update for jobID %s buildID %s to lambdatest %+v", jobID, buildID, resourceStats)
	resourceStatsMessage := CreateResourceStatsMessage(resourceStats)
	s.writeMessageToBuffer(&resourceStatsMessage)
}

// writeMessageToBuffer  writes all message to buffer channel
func (s *synapse) writeMessageToBuffer(message *core.Message) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		s.logger.Errorf("error marshaling message")
		return
	}
	s.MsgChan <- messageJSON
}

// messageWriter writes the messages to open websocket
func (s *synapse) messageWriter(conn *websocket.Conn) {
	for {
		select {
		case <-s.ConnectionAborted:
			return
		case messageJson := <-s.MsgChan:
			if err := conn.WriteMessage(websocket.TextMessage, messageJson); err != nil {
				s.logger.Errorf("error sending message to the server error %v", err)
				s.MsgChan <- messageJson
				s.MsgErrChan <- struct{}{}
				close(s.MsgErrChan)
				return
			}
		}
	}
}

func (s *synapse) processYMLParsingRequest(message core.Message) {
	var parsingReqMsg *core.YMLParsingRequestMessage
	var writeMsg core.Message
	defer s.writeMessageToBuffer(&writeMsg)
	if err := json.Unmarshal(message.Content, &parsingReqMsg); err != nil {
		s.logger.Errorf("error in unmarshaling message for yml parsing request, error %v ", err)

		writeMsg = createYMlParsingResultMessage(core.YMLParsingResultMessage{
			OrgID:    parsingReqMsg.OrgID,
			BuildID:  parsingReqMsg.BuildID,
			ErrorMsg: err.Error(),
		})
		return
	}
	oauth := s.secretsManager.GetOauthToken()

	tasOutput, err := s.tasConfigDownloader.GetTASConfig(context.TODO(), parsingReqMsg.GitProvider,
		parsingReqMsg.CommitID,
		parsingReqMsg.RepoSlug, parsingReqMsg.TasFileName, oauth,
		parsingReqMsg.Event, parsingReqMsg.LicenseTier)
	if err != nil {
		s.logger.Errorf("error occurred while fetching tas config file for buildID %s orgID %s, error %v",
			parsingReqMsg.BuildID, parsingReqMsg.OrgID, err)
		writeMsg = createYMlParsingResultMessage(core.YMLParsingResultMessage{
			OrgID:    parsingReqMsg.OrgID,
			BuildID:  parsingReqMsg.BuildID,
			ErrorMsg: err.Error(),
		})
		return
	}
	writeMsg = createYMlParsingResultMessage(core.YMLParsingResultMessage{
		OrgID:     parsingReqMsg.OrgID,
		BuildID:   parsingReqMsg.BuildID,
		YMLOutput: *tasOutput,
	})
}
