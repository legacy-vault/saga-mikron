// chat.go

/*

	Web Chat «SAGA MIKRON».

	Version: 0.4.1.
	Date of Creation: 2017-05-03.
	Author: McArcher.

	This is a simple web Chat.

	The Client is embedded into the Server, that means:	all you need to do is
	to configure the Server; and then Clients will see Results in their web
	Browsers.

	The Name of the Chat is the Memory of previous Civilizations on the Planet
	Earth. The word "Mikron" is taken from the Greek Language. The Word "Saga"
	is the Word which means "a Tale", something that people said to each other.
	The English word "say" origins from that ancient Word "Saga".

*/

//------------------------------------------------------------------------------

package main

import (
	"flag"
	"log"
	"math"
	"math/rand"
	"time"
)

//------------------------------------------------------------------------------

// Lists
type tChatRecord struct {
	time    int64  // Post Time, Unix Timestamp
	author  uint64 // UID of the Author
	message string // Message
}
type tChatRecords [chat_recordsMaxLast + 1]tChatRecord

type tChatJob struct {
	chatRecord    tChatRecord
	returnChannel chan tChatJob
}

type tLoginJob struct {
	client        tActiveClient
	uid           uint64
	result        bool
	returnChannel chan tLoginJob
}

type tRegisterJob struct {
	name, pwd     string
	uid           uint64
	returnChannel chan tRegisterJob
	result        bool
}

//------------------------------------------------------------------------------

const chat_systemUserUID uint64 = 0         // UID of the Chat's System User
const chat_systemUserName string = "SYSTEM" // Name of the Chat's System User
const chatJobBufferLength = 64              // Buffer Length of the Chat Jobs Channel
const loginManagerChanBufferLen = 64        // Buffer Length of the Login Manager's Channel
const registerManagerChanBufferLen = 64     // Buffer Length of the Register Manager's Channel

// Size Limits
const chat_recordsMaxLast = math.MaxUint16 // Maximum Number of the Last Chat Message
// This Limit can not be different from uint's Limit, as we need an Overflow to
// be present to simulate the endless List (Array)

//------------------------------------------------------------------------------

// Flags
var flag_port_ptr = flag.String("port", srv_port_default, "Port Number.")
var flag_ipAddress_ptr = flag.String("ip", srv_ipAddress_default, "IP Address.")

var flag_userDataFile_ptr = flag.String("udf", file_userData_default,
	"Path to User Data File.")

var flag_createUserDataFile_ptr = flag.Bool("cudf", false,
	"Create User Data File if it does not exist.")

var flag_indexFile_ptr = flag.String("if", file_index_default,
	"Path to Index File Template.")

var flag_chatFile_ptr = flag.String("cf", file_chat_default,
	"Path to Chat File Template.")

var flag_userRegdFile_ptr = flag.String("urf", file_userRegistered_default,
	"Path to 'User Registered' File Template.")

var flag_ari_ptr = flag.Int("ari", activityRevisorInterval_default,
	"Activity Revisor Interval, in Seconds.")

var flag_asqRevInt_ptr = flag.Int("asqri", asq_revisor_interval_default,
	"Anti-Spam Questions Revisor Interval, in Seconds.")

// Lists
var chatRecordsList tChatRecords

// Internal Parameters
var chat_recordFirstNum uint16 // Index of the Firts actual Element in List (Array)
var chat_recordLastNum uint16  // Index of the Last actual Element in List (Array)
// The Counters can not be different from un-signed Integer Type (uint8, uint16,
// ...), as we need an Overflow to be present to simulate the endless List.

var chat_recordFirstTimestamp int64 // Timestamp of the First actual Element in List (Array)
var chat_recordLastTimestamp int64  // Timestamp of the Last actual Element in List (Array)
// It may first seem that Timestamps are a waste of Resources, but it is not.
// If by the means of an Accident a Client loses connection to the Server, and
// the Server's Sessioun Timeout Parameter is set to a large Value, and at the
// same Time an extremely great Activity starts in Chat, the "lmid" of a User
// may become literally outdated when new Flood of Messages makes a fuul Circle
// in the List and re-writes the last seen Message. In such Case, Timestamps
// can help such Client (when he fixes his Network Connection) to partially
// restore the Messages which he has missed.

var firstCircle bool // Shows whether any Overflow happened or not

// Channels
var chatManagerChan chan tChatJob
var loginManagerChan chan tLoginJob
var registerManagerChan chan tRegisterJob
var chatQuit, chatManagerQuit, loginManagerQuit, registerManagerQuit chan int

//------------------------------------------------------------------------------

func main() {

	// Main Function.

	var ok bool = false

	// Preparations
	flags_init()
	chat_init()

	// Templates
	ok = templates_init()
	if !ok {
		return
	}

	// User Data
	ok = userData_init()
	if !ok {
		return
	}

	saveRegisteredUsersToTpl() // Must be run after userData_init() !

	// Server
	server.ipAddress = srv_ipAddress
	server.port = srv_port
	server.init()
	server.start()

	<-chatQuit // Wait for Signal from Channel
	server.stop()
}

//------------------------------------------------------------------------------

func flags_init() {

	// Initializes the Flags, id est, Command Line Parameters of this Program.

	flag.Parse()

	// Server
	srv_port = *flag_port_ptr
	srv_ipAddress = *flag_ipAddress_ptr

	// Files
	createUserDataFile = *flag_createUserDataFile_ptr
	file_userData = *flag_userDataFile_ptr
	file_indexTemplate = *flag_indexFile_ptr
	file_chatTemplate = *flag_chatFile_ptr
	file_userRegdTemplate = *flag_userRegdFile_ptr

	// Revisors
	activityRevisorInterval = *flag_ari_ptr
	asq_revisorInterval = *flag_asqRevInt_ptr
}

//------------------------------------------------------------------------------

func chat_init() {

	// Initializes the Chat.

	var seed int64

	// Random Number Generator
	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	chatQuit = make(chan int)

	// Initial Values of Counters
	firstCircle = true
	chat_recordFirstNum = 0
	chat_recordFirstTimestamp = time.Now().Unix()

	// First Message ever
	chat_recordLastNum = 0
	chat_recordLastTimestamp = time.Now().Unix()
	chatRecordsList[chat_recordLastNum].message = "Chat Server started."
	chatRecordsList[chat_recordLastNum].time = time.Now().Unix()
	chatRecordsList[chat_recordLastNum].author = chat_systemUserUID
}

//------------------------------------------------------------------------------

func chatManager() {

	// Manages incoming Messages.

	var loop bool = true
	var job tChatJob
	var now int64

	for loop {

		job = <-chatManagerChan // Get Job from Channel

		now = time.Now().Unix()
		job.chatRecord.time = now

		// Manipulate Counters & put Message into List
		if chat_recordLastNum == chat_recordsMaxLast {
			firstCircle = false
		}

		chat_recordLastNum++ // Automatic Overflow makes it "endless"

		if firstCircle { // First Circle

			// Change only last Element
			chat_recordLastTimestamp = now

		} else { // Circle #2, #3, ...

			// Change both Elements
			chat_recordFirstNum = chat_recordLastNum + 1
			chat_recordLastTimestamp = now
			chat_recordFirstTimestamp = chatRecordsList[chat_recordFirstNum].time

		}
		chatRecordsList[chat_recordLastNum] = job.chatRecord

		job.returnChannel <- job // Send back

		// Checking for Stop Signal
		select {
		case <-chatManagerQuit:
			loop = false
			log.Println("Closing Chat Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------

func loginManager() {

	// Manages Logging-In Requests.

	var loop bool = true
	var job tLoginJob
	var logged bool
	var activeClient *tActiveClient
	var rcvChan chan tActiveJob // for Requests to activeManager
	var activeJob *tActiveJob   // for Requests to activeManager
	var now int64

	// Preparations
	activeClient = new(tActiveClient)
	rcvChan = make(chan tActiveJob)         // for Requests to activeManager
	activeJob = new(tActiveJob)             // ~
	activeJob.action = activeJobUpdateCache // Update Cache
	activeJob.returnChannel = rcvChan       // ~

	for loop {

		job = <-loginManagerChan // Get Job from Channel

		// Check (once again) that Client is not logged in
		_, logged = activeClientsList[job.uid]
		if logged {

			job.result = false
			job.returnChannel <- job // Send back

		} else {

			now = time.Now().Unix()

			// Create new Active Client
			activeClient.address = job.client.address
			activeClient.sid = job.client.sid

			// This Value will then be updated by the activeManager
			activeClient.lastActiveTime = now

			// This is the Timestamp of Log-In Event and stays constant.
			// This Value is snapped to a nearest previous Message's Timestamp.
			// But even if you do not snap it, Server will automatically
			// synchronize not accurate Timestamps.
			activeClient.log_ts = chatRecordsList[chat_recordLastNum].time

			activeClient.log_mid = chat_recordLastNum // 0 at Server's Start

			// Update List
			activeClientsList[job.uid] = *activeClient // it is thread-safe
			// Notes:
			// Only the "loginManager" can add new Clients to the  List.
			// As we are sure that UID is unique (not logged-in), it is not read
			// by anone else. "activeManager" can modify only existing active
			// Clients.

			job.result = true
			job.returnChannel <- job // Send back

			// After adding new Client to the List, we must refresh the cached
			// List of active Clients. While Deletion and Modification of
			// active Clients is done by "activeManager", we send Signal to him.

			// Send Job
			activeManagerChan <- *activeJob

			// Get Feedback
			*activeJob = <-rcvChan

		}

		// Checking for Stop Signal
		select {
		case <-loginManagerQuit:
			loop = false
			log.Println("Closing Chat Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------

func registerManager() {

	// Manages Register Requests.

	var loop bool = true
	var job tRegisterJob

	for loop {

		job = <-registerManagerChan // Get Job from Channel

		job.result, job.uid = user_register(&job.name, &job.pwd)

		job.returnChannel <- job // Send back

		// Checking for Stop Signal
		select {
		case <-registerManagerQuit:
			loop = false
			log.Println("Closing Register Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------
