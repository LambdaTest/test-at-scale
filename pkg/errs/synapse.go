package errs

import (
	"fmt"
	"strings"
)

// ERR_DUMMY dummy error
var ERR_DUMMY = Err{
	Code:    "ERR::DUMMY",
	Message: "Dummy error "}

// ERR_INVALID_ENVIRONMENT  should be thorwn when invalid environment specified"
var ERR_INVALID_ENVIRONMENT = Err{
	Code:    "ERR::INV::ENV",
	Message: "Invalid environment specified"}

// ERR_CTRL_CONN_MAX_ATTEMPT should be thrown when control websocket reconnection max attempt reached
var ERR_CTRL_CONN_MAX_ATTEMPT = Err{
	Code:    "ERR::CTRL::CONN::MAX::ATTEMPT",
	Message: "Control websocket reconnection max attempt reached"}

// ERR_SNK_PRX_MAX_ATTEMPT should be thrown when sink proxy restart max attempt reache
var ERR_SNK_PRX_MAX_ATTEMPT = Err{
	Code:    "ERR::SNK::PRX::MAX::ATTEMPT",
	Message: "Sink proxy restart max attempt reached"}

// ERR_INF_API_MAX_ATTEMPT should be thrown when info api server restart max attempt reached
var ERR_INF_API_MAX_ATTEMPT = Err{
	Code:    "ERR::INF::API::MAX::ATTEMPT",
	Message: "Info api server restart max attempt reached"}

// ERR_FS_MAX_ATTEMPT should be thrown when file server restart max attempt reached
var ERR_FS_MAX_ATTEMPT = Err{
	Code:    "ERR::FS::MAX::ATTEMPT",
	Message: "File server restart max attempt reached"}

// ERR_INV_WS_DAT_TYPE should be thrown when invalid data type reader received from websocket
var ERR_INV_WS_DAT_TYPE = Err{
	Code:    "ERR::INV::WS::DAT::TYPE",
	Message: "Invalid data type reader received from websocket"}

// ERR_BIN_UPD  function retruns err with code  "ERR::BIN::UPD"
func ERR_BIN_UPD(err string) Err {
	return Err{
		Code:    "ERR::BIN::UPD",
		Message: "Unable to update binary " + err}
}

// ERR_WS_CTRL_CONN function returns err with code  "ERR::WS::Conn"
func ERR_WS_CTRL_CONN(err string) Err {
	return Err{
		Code:    "ERR::WS::Conn",
		Message: "Unable to establish control websocket connection " + err}
}

// ERR_WS_CONN function returns err with code  "ERR::WS::Conn"
func ERR_WS_CONN(err string) Err {
	return Err{
		Code:    "ERR::WS::Conn",
		Message: "Unable to  establish websocket connection " + err}
}

// ERR_WS_CTRL_CONN_DWN function returns err with code "ERR::WS::CTRL::CONN::DWN"
func ERR_WS_CTRL_CONN_DWN(err string) Err {
	return Err{
		Code:    "ERR::WS::CTRL::CONN::DWN",
		Message: "Control websocket connection closed " + err}
}

// ERR_DAT_CONN_DWN function returns err with code "ERR::DAT::CONN::DWN"
func ERR_DAT_CONN_DWN(err string) Err {
	return Err{
		Code:    "ERR::DAT::CONN::DWN",
		Message: "Data websocket connection closed " + err}
}

// ERR_INVALID_WS_URL function returns err with code "ERR::INV::WS::URL"
func ERR_INVALID_WS_URL(err string) Err {
	return Err{
		Code:    "ERR::INV::WS::URL",
		Message: "Invalid websocket url error " + err}
}

// ERR_SNK_PRX function return error with code "ERR::SNK::PRX"
func ERR_SNK_PRX(err string) Err {
	return Err{
		Code:    "ERR::SNK::PRX",
		Message: "Sink proxy failed :  " + err}
}

// ERR_SNK_PRX_CONN function returns error with code "ERR::SNK::PRX::CONN"
func ERR_SNK_PRX_CONN(err string) Err {
	return Err{
		Code:    "ERR::SNK::PRX::CONN",
		Message: "Unable to establish connection to local proxy :  " + err}
}

// ERR_WS_WRT function returns error with code "ERR::WS::WRT"
func ERR_WS_WRT(err string) Err {
	return Err{
		Code:    "ERR::WS::WRT",
		Message: "Unable to valid retrieve writer from ws :  " + err}
}

// ERR_WS_RDR function returns error with code "ERR::WS::RDR"
func ERR_WS_RDR(err string) Err {
	return Err{
		Code:    "ERR::WS::RDR",
		Message: "Unable to retrieve valid reader from ws :  " + err}
}

// ERR_ATT_PRX function returns error with code "ERR::ATT::PRX"
func ERR_ATT_PRX(reqType string, err string) Err {
	return Err{
		Code:    "ERR::ATT::PRX",
		Message: fmt.Sprintf("Unable to attach proxy to [ %s ]request :  %s", reqType, err)}
}

// ERR_DNS_RLV function returns error with code  "ERR::DNS::RLV"
func ERR_DNS_RLV(err string) Err {
	return Err{
		Code:    "ERR::DNS::RLV",
		Message: fmt.Sprintf("Error while resolving dns :  %s", err)}
}

// ERR_VLD_CFG function return error with code ERR::CNF::FLD::VLD
func ERR_VLD_CFG(errs []string) Err {
	return Err{
		Code:    "ERR::CNF::FLD::VLD",
		Message: fmt.Sprintf("Validation errors :  \n%s", strings.Join(errs, "\n"))}
}

// ERR_DAT_WS_RD function returns error with code ERR::DAT::WS::RD
func ERR_DAT_WS_RD(err string) Err {
	return Err{
		Code:    "ERR::DAT::WS::RD",
		Message: fmt.Sprintf("Unable to read from websocket :  \n%s", err)}
}

// ERR_SNK_WRT function returns error with code ERR::SNK::WRT
func ERR_SNK_WRT(err string) Err {
	return Err{
		Code:    "ERR::SNK::WRT",
		Message: fmt.Sprintf("Unable to read from websocket :  \n%s", err)}
}

// ERR_API_SRV_STR function returns error with code ERR::API::SRV::STR
func ERR_API_SRV_STR(err string) Err {
	return Err{
		Code:    "ERR::API::SRV::STR",
		Message: fmt.Sprintf("Unable to start api server :  \n%s", err)}
}

// ERR_FIL_SRV_STR function returns error with code "ERR::FIL::SRV::STR"
func ERR_FIL_SRV_STR(err string) Err {
	return Err{
		Code:    "ERR::FIL::SRV::STR",
		Message: fmt.Sprintf("Unable to start file server :  \n%s", err)}
}

// ERR_DIR_CRT function returns error with code "ERR::DIR::CRT"
func ERR_DIR_CRT(err string) Err {
	return Err{
		Code:    "ERR::DIR::CRT",
		Message: fmt.Sprintf("Unable to create directory :  \n%s", err)}
}

// ErrDirDel function returns error with code "ERR::DIR::DEL"
func ErrDirDel(err string) Err {
	return Err{
		Code:    "ERR::DIR::DEL",
		Message: fmt.Sprintf("Unable to delete directory :  \n%s", err)}
}

// ERR_FIL_CRT function returns error with code ERR::FIL::CRT
func ERR_FIL_CRT(err string) Err {
	return Err{
		Code:    "ERR::FIL::CRT",
		Message: fmt.Sprintf("Unable to create file :  \n%s", err)}
}

// ERR_API_WEB_HOK function returns error with code ERR::API::WEB::HOK
func ERR_API_WEB_HOK(err string) Err {
	return Err{
		Code:    "ERR::API::WEB::HOK",
		Message: fmt.Sprintf("Unable to call webhook url :  \n%s", err)}
}

// ERR_DOCKER_RUN function returns error with code ERR::DOCKER::RUN
func ERR_DOCKER_RUN(err string) Err {
	return Err{
		Code:    "ERR::DOCKER::RUN",
		Message: fmt.Sprintf("Docker run failed with error:  \n%s", err)}
}

// ERR_DOCKER_CRT function returns error with code ERR::DOCKER::CRT
func ERR_DOCKER_CRT(err string) Err {
	return Err{
		Code:    "ERR::DOCKER::CRT",
		Message: fmt.Sprintf("Docker create failed with error:  \n%s", err)}
}

// ERR_DOCKER_STRT function returns error with code "ERR::DOCKER::STRT"
func ERR_DOCKER_STRT(err string) Err {
	return Err{
		Code:    "ERR::DOCKER::STRT",
		Message: fmt.Sprintf("Docker start failed with error:  \n%s", err)}
}

// ErrDockerVolCrt function returns error with code "ERR::DOCKER::VOL::CRT"
func ErrDockerVolCrt(err string) Err {
	return Err{
		Code:    "ERR::DOCKER::VOL::CRT",
		Message: fmt.Sprintf("Docker volume create failed with error:  \n%s", err)}
}

// ErrDockerCP function returns error with code "ERR::DOCKER::CP"
func ErrDockerCP(err string) Err {
	return Err{
		Code:    "ERR::DOCKER::CP",
		Message: fmt.Sprintf("Error copying file to docker:  \n%s", err)}
}

// ErrSecretLoad function returns error with code "ERR::SECRET::LOAD"
func ErrSecretLoad(err string) Err {
	return Err{
		Code:    "ERR::SECRET::LOAD",
		Message: fmt.Sprintf("Error in loading secrets:  \n%s", err)}
}

// ERR_JSON_MAR function returns error with code "ERR::JSON::MAR"
func ERR_JSON_MAR(err string) Err {
	return Err{
		Code:    "ERR::JSON::MAR",
		Message: fmt.Sprintf("Error marshaling JSON:  \n%s", err)}
}

// ERR_JSON_UNMAR function returns error with code "ERR::JSON::UNMAR"
func ERR_JSON_UNMAR(err string) Err {
	return Err{
		Code:    "ERR::JSON::UNMAR",
		Message: fmt.Sprintf("Error unmarshaling JSON:  \n%s", err)}
}

// ERR_LT_CRDS functio returns error with code "ERR::LT::CRDS"
func ERR_LT_CRDS() Err {
	return Err{
		Code:    "ERR::LT::CRDS",
		Message: "No lambdatest config provided"}
}

// ERR_SNK_RD_WRT_MSM should be raise when there is read write mismatch in sink proxy
var ERR_SNK_RD_WRT_MSM = Err{
	Code:    "ERR::SNK::RD::WRT::MSM",
	Message: "Read write mismatch in sink proxy "}

// CR_AUTH_NF should be raise when container registry auth are not present for private repo
var CR_AUTH_NF = Err{
	Code:    "CR::AUTH:NF",
	Message: "Container registry auth are not present for private repo"}
