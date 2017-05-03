// server.go

package main

import (
	"log"
	"net/http"
	"time"
)

//------------------------------------------------------------------------------

// Server
type tServer struct {
	ipAddress string
	port      string
	server    http.Server
}

type tServerJob struct {
	actionType    uint8
	w             http.ResponseWriter
	req           *http.Request
	returnChannel chan tServerJob
}

// Actions
type tActions [srv_actionsCount]func(http.ResponseWriter, *http.Request)

//------------------------------------------------------------------------------

// Server
const srv_port_default = "2000"         // Default Port of the Server
const srv_ipAddress_default = "0.0.0.0" // Default IP Address of the Server
const srv_protocol = "http://"          // Protocol of the Server

// Actions
const srv_actionsCount = 10 // Possible Actions to do with the Client's Request

// Client Behaviour
const redirectDelay_str = "0"       // Delay of Page Redirect, in Seconds
const sendToGetDelay_str = "1"      // Delay between sent Message and getting Updates, in Seconds
const msgUpdateInterval_str = "10"  // Interval between last and next Update Requests for New Messages
const userUpdateInterval_str = "45" // Interval between last and next Update Requests for User List

// URL Path
const path_index = "/"       // Path to Index Page (may differ from Root!)
const path_register = "/r"   // Registration Page
const path_login = "/l"      // Log-In Page
const path_logout = "/x"     // Log-Out Page
const path_chat = "/c"       // Chat's main Page
const path_news = "/d"       // Page for checking new Messages from Server
const path_send = "/s"       // Page for sending Messages to Server
const path_activeList = "/a" // Page for List of active Users
const path_stat = "/t"       // Statistics Page
const path_asq = "/q"        // Path for requesting Anti-Spam Question

// Server's Reply Codes
const code_messageSent = "O"  // Server's Reply if Message is sent
const code_NotLoggedIn = "L"  // Server's Reply if User is Not Logged In or Idle
const code_EmptyMessage = "E" // Server's Reply if User is sending an empty Message
const code_BadPOSTdata = "X"  // Server's Reply if Error in POST Data
const code_BadRequest = "B"   // Server's Reply if Error in LMS Parameter
const code_NoNews = "N"       // Server's Reply if No New Messages Found
const code_msgTooLong = "M"   // Server's Reply if Client's Message is too long

// Client's HTML Form Parameter Names, POST/GET Variable Names
const param_login_userID = "luid" // UID during Logging-In
const param_login_password = "lp" // Password during Logging-In
const param_reg_userName = "rn"   // Name during Registration
const param_reg_password = "rp"   // Pasword during Registration
const param_qid = "qid"           // ID of Anti-Spam Question
const param_qAnswer = "qa"        // Answer to an Anti-Spam Question
const param_unknownVal = "X"      // Such Value shows that Client does not know his Parameter
const param_req_mid = "mid"       // ID of last Message known
const param_req_ts = "ts"         // Last known Timestamp

// Size Limits
const userName_maxLen = 255       // Maximum Length of the Name for Registration
const userPwd_maxLen = 255        // Maximum Length of the Password for Registration
const msgMaxSize = 4096           // Maximum Size of Message sent from Client, in Bytes
const serverJobsBufferSize = 1024 // Size of the Buffer of the Channel for Jobs

//------------------------------------------------------------------------------

// Server
var server tServer
var srv_port, srv_ipAddress string
var action tActions

// Channels
var serverJobsChan chan tServerJob
var serverJobsManagerQuit chan int

//------------------------------------------------------------------------------

func (srv *tServer) init() {

	// Configures the Server.

	// Server Port & Address
	srv.server.Addr = srv.ipAddress + ":" + srv.port
	srv.server.IdleTimeout = 30 * time.Second
	http.Handle("/", http.HandlerFunc(httpHandler))

	// Actions, Array of "Pointers" to Functions
	action[0] = page_delta
	action[1] = page_send
	action[2] = page_activeList
	action[3] = page_index
	action[4] = page_chat
	action[5] = page_stat
	action[6] = page_login
	action[7] = page_logout
	action[8] = page_register
	action[9] = page_asq

	// Server Manager
	serverJobsChan = make(chan tServerJob, serverJobsBufferSize)
	serverJobsManagerQuit = make(chan int)

	// Active Revisor & Active Clients List
	activeClientsList = make(tActiveClients)
	activeRevisorQuit = make(chan int)

	// Active Manager
	activeManagerChan = make(chan tActiveJob, activeManagerChanBufferLen)
	activeManagerQuit = make(chan int)

	// ASQ Revisor
	asqsList = make(tAntiSpamQuestions)
	asqRevisorQuit = make(chan int)

	// ASQ Manager
	asqManagerChan = make(chan tAsqJob, asqManagerChanBufferLen)
	asqManagerQuit = make(chan int)

	// Chat Manager
	chatManagerChan = make(chan tChatJob, chatJobBufferLength)
	chatManagerQuit = make(chan int)

	// Log-In Manager
	loginManagerChan = make(chan tLoginJob, loginManagerChanBufferLen)
	loginManagerQuit = make(chan int)

	// Register Manager
	registerManagerChan = make(chan tRegisterJob, registerManagerChanBufferLen)
	registerManagerQuit = make(chan int)
}

//------------------------------------------------------------------------------

func (srv *tServer) start() {

	// Starts the Server.

	// Server
	go srv.startServerRoutine()

	// Server Manager
	go serverJobsManager()

	// Active Revisor
	go activeRevisor()

	// Active Manager
	go activeManager()

	// ASQ Revisor
	go asqRevisor()

	// ASQ Manager
	go asqManager()

	// Chat Manager
	go chatManager()

	// Log-In Manager
	go loginManager()

	// Register Manager
	go registerManager()
}

//------------------------------------------------------------------------------

func (srv *tServer) startServerRoutine() {

	// Starts a Listener.

	var err error

	log.Println("Server started at", srv.server.Addr) //
	err = srv.server.ListenAndServe()
	if err != nil {
		log.Println("Server Error:", err) //
		return
	}
}

//------------------------------------------------------------------------------

func (srv *tServer) stop() {

	// Shuts the Server down.

	var err error

	err = srv.server.Shutdown(nil)
	if err != nil {
		log.Println("Error during Server Shutdown:", err) //
		return
	}

	// Stop Go-Routines
	serverJobsManagerQuit <- 1
	activeRevisorQuit <- 1
	activeManagerQuit <- 1
	asqRevisorQuit <- 1
	asqManagerQuit <- 1
	chatManagerQuit <- 1
	loginManagerQuit <- 1
	registerManagerQuit <- 1

	log.Print("Server Stopped.\n\n") //
}

//------------------------------------------------------------------------------

func httpHandler(w http.ResponseWriter, req *http.Request) {

	// Processes and serves the HTTP Request from Client.

	var actionNum uint8
	var rcvChan chan tServerJob
	var job *tServerJob

	switch req.URL.Path {

	case path_news:
		actionNum = 0

	case path_send:
		actionNum = 1

	case path_activeList:
		actionNum = 2

	case path_index:
		actionNum = 3

	case path_chat:
		actionNum = 4

	case path_stat:
		actionNum = 5

	case path_login:
		actionNum = 6

	case path_logout:
		actionNum = 7

	case path_register:
		actionNum = 8

	case path_asq:
		actionNum = 9

	default:
		actionNum = 3 // page_index
	}

	// Creating a Job
	rcvChan = make(chan tServerJob) // Channel, from which Reply comes to us
	job = new(tServerJob)
	job.actionType = actionNum
	job.req = req
	job.w = w
	job.returnChannel = rcvChan // Writing "return-to Address"

	// Send a Job
	serverJobsChan <- *job

	// Wait for Reply
	*job = <-rcvChan
}

//------------------------------------------------------------------------------

func serverJobsManager() {

	// Manages incoming Server Jobs.

	var loop bool = true
	var job *tServerJob

	job = new(tServerJob)

	for loop {

		*job = <-serverJobsChan                // Get Job from Channel
		action[job.actionType](job.w, job.req) // Do Action
		job.returnChannel <- *job              // Send back

		// Checking for Stop Signal
		select {
		case <-serverJobsManagerQuit:
			loop = false
			log.Println("Closing Jobs Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------
