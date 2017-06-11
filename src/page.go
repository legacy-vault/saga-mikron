// page.go

package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

//------------------------------------------------------------------------------

func page_delta(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request of Delta Page.
	// Gives Client a Portion of new Messages since Clients's previous Request.

	// Client sends a Request as a 'application/x-www-form-urlencoded'.
	// A standard Parser decodes the Request.

	// Server replies to client one of the following:
	//		1. code_NotLoggedIn ('L'),
	//		2. code_BadPOSTdata ('X'),
	//		3. code_BadRequest ('B'),
	//		4. JSON (new_messages),
	//		5. nothing ('').

	var req_mid uint16 // Requested "mid"
	// ID of the last seen Message or of the last Message before Log-In

	var req_ts int64 // Requested "ts"
	// Timestamp of the last seen Event (Message or Log-In)

	var log_mid uint16 // Client's "mid"
	var log_ts int64   // Client's "ts"

	var i uint16
	var ok bool
	var err, err2 error
	var req_mid_str, req_ts_str string
	var req_mid_uint64, req_ts_uint64 uint64

	var author, time_str, msg string
	var outMsgFirst, outMsgLast uint16 // Indexes of Messages which to give the Client

	// Correct Cookies & Not Idle ?  & update User's Last Activity Time
	ok, _, log_mid, log_ts = user_check(w, req)

	// Logged in ?
	if !ok {
		fmt.Fprint(w, code_NotLoggedIn) // Not Logged In
		return
	}

	// Reading Client's Request
	err = req.ParseForm()
	if err != nil {
		log.Println("Error Reading POST Form:", err) //
		fmt.Fprint(w, code_BadPOSTdata)              // POST Error
		ok = false
		return
	}
	req_mid_str = req.PostFormValue(param_req_mid)
	req_ts_str = req.PostFormValue(param_req_ts)

	// Empty LMS
	if (len(req_mid_str) == 0) || (len(req_ts_str) == 0) {
		log.Println("Empty Request.")  //
		fmt.Fprint(w, code_BadRequest) // Bad Request
		return
	}

	// Known or un-Known ?
	if (req_mid_str == param_unknownVal) || (req_ts_str == param_unknownVal) { // 'X'

		// If Client does not know, then tell him Values (No Messages are sent).
		fmt.Fprint(w, "{\"messages\":[], \"x\":{\"", param_req_mid, "\":\"",
			log_mid, "\", \"", param_req_ts, "\":\"", log_ts, "\"} }")
		return
	}

	// Requested "mid" and "ts" are known, are supposed to be numeric
	req_mid_uint64, err = strconv.ParseUint(req_mid_str, 10, 64)
	req_ts_uint64, err2 = strconv.ParseUint(req_ts_str, 10, 64)
	if (err != nil) || (err2 != nil) {
		log.Println("Bad Request:", err, err2) //
		fmt.Fprint(w, code_BadRequest)         // Bad Request
		return
	}
	req_mid = uint16(req_mid_uint64)
	req_ts = int64(req_ts_uint64)

	// Any News?
	if chat_recordLastTimestamp < req_ts {

		fmt.Fprint(w, code_NoNews) // No News
		return
	}

	if req_mid > chat_recordsMaxLast { // no Sub-Zero Check as uint is always > 0.

		log.Println("Bad Request: req_mid is out of Range.") //
		fmt.Fprint(w, code_BadRequest)                       // Bad Request
		return
	}

	if req_ts < chatRecordsList[log_mid].time {
		log.Println("Bad Request: req_ts is out of Range.") //
		fmt.Fprint(w, code_BadRequest)                      // Bad Request
		return
	}

	if req_ts < chat_recordFirstTimestamp {

		// Client has been sleeping too long or new Circle of Messages has re-written old Messages
		outMsgFirst = chat_recordFirstNum
		outMsgLast = chat_recordLastNum

	} else {

		// Message #mid was earlier than
		if req_ts != chatRecordsList[req_mid].time {

			// Client is non-synchronized or crazy. Or it is a cool h4X0R...
			// We do not reject even crazy Clients :D
			log.Println("Synchronizing crazy Client...") //
			fmt.Fprint(w, "{\"messages\":[], \"x\":{\"", param_req_mid, "\":\"",
				req_mid, "\", \"", param_req_ts, "\":\"",
				chatRecordsList[req_mid].time, "\"} }")
			return
		}

		// Simple Situation
		outMsgFirst = req_mid + 1
		outMsgLast = chat_recordLastNum

	}

	if outMsgLast < outMsgFirst {

		fmt.Fprint(w, code_NoNews) // No News
		return
	}

	// Writing in JSON Format
	/*
		{
		 "messages":
				[
					{"mid":"123", "tim":"00", "atr":"Вася", "txt":"AU8Xv745cd=="},
					{"mid":"124", "tim":"00", "atr":"Петя", "txt":"BU8Xv745cd=="},
					{"mid":"125", "tim":"00", "atr":"Коля", "txt":"CU8Xv745cd=="}
				],
		 "x":
				{"mid":"125", "ts":"1234567"}
		}
	*/

	fmt.Fprint(w, "{\"messages\": [")

	/*

		Notes:

		1. It is thread-safe to read "chatRecordsList" because even after Chat-
		Reset it is available for Read. The only Condition for it to be bad is
		when it gets re-written by next Message with the same MID after the
		Reset. To make it happen a lot of Time must pass (Duration of full
		Circle from Message #N to same Message #N. It may take Hours or may even
		not happen if Server gets rebooted. Of course, if the Client is set up
		to ask for new Messages once in a Year, it may happen... but in
		this case this Web Chat has no Sense at all :D

		2. "userDataList" is thread-safe to read because it is not modified.

	*/

	i = outMsgFirst
	for {

		// Write all except the last Message
		if i >= outMsgLast {
			break
		}

		time_str = time.Unix(chatRecordsList[i].time, 0).Format("15:04:05")
		author = base64.StdEncoding.EncodeToString([]byte(userDataList[chatRecordsList[i].author].name))
		msg = base64.StdEncoding.EncodeToString([]byte(chatRecordsList[i].message))

		fmt.Fprintf(w, "{\"mid\":\"%d\",\"tim\":\"%s\",\"atr\":\"%s\",\"txt\":\"%s\"},",
			i, time_str, author, msg)

		i++

	}

	// Write last Message; last element, without ","
	if i == outMsgLast {

		time_str = time.Unix(chatRecordsList[i].time, 0).Format("15:04:05")
		author = base64.StdEncoding.EncodeToString([]byte(userDataList[chatRecordsList[i].author].name))
		msg = base64.StdEncoding.EncodeToString([]byte(chatRecordsList[i].message))

		fmt.Fprintf(w, "{\"mid\":\"%d\",\"tim\":\"%s\",\"atr\":\"%s\",\"txt\":\"%s\"}",
			i, time_str, author, msg)
	}

	// Updated "mid" & "ts"
	fmt.Fprint(w, "], \"x\":{\"", param_req_mid, "\":\"",
		chat_recordLastNum, "\", \"", param_req_ts, "\":\"", chat_recordLastTimestamp, "\"} }")
}

//------------------------------------------------------------------------------

func page_send(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request to send a Message to Chat.

	/*
		Client sends a Request as a 'text/plain; charset=utf-8'.
		Format of the Request:
			<Text_A> <One Space Symbol> <Text_B>.
		Here <Text_A> is Letters Count of <Text_B>.
		Examples:
			'15 This is a test.', '6 Hello!'.
	*/
	// Server replies to client one of the following:
	//		1. code_NotLoggedIn ('L')
	//		2. code_POSTdata ('X')
	//		3. code_EmptyMessage ('E')
	//		4. code_OK ('O')
	//		5. ...

	var ok bool
	var uid uint64
	var reqBody []byte
	var err error
	var reqBody_str, p1, p2, txt_safe string
	var spaceIndex int
	var p1_int64 int64
	var chatJob *tChatJob
	var rcvChan chan tChatJob

	// Correct Cookies & Not Idle ?  & update User's Last Activity Time
	ok, uid, _, _ = user_check(w, req)
	if !ok {
		fmt.Fprint(w, code_NotLoggedIn) // Error: Not Logged In or Idle
		return
	}

	// Reading Client's message
	reqBody, err = ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		fmt.Fprint(w, code_BadPOSTdata) // Error in Data
		return
	}

	// Empty Body ?
	if reqBody == nil {
		log.Printf("nil body. header=[%s].", req.Header) //
		fmt.Fprint(w, code_BadPOSTdata)                  // Error in Data
	}

	// Decoding Contents
	reqBody_str = string(reqBody)
	spaceIndex = strings.Index(reqBody_str, " ")
	p1 = reqBody_str[0:spaceIndex]
	p2 = reqBody_str[spaceIndex+1:]
	p1_int64, err = strconv.ParseInt(p1, 10, 64)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		fmt.Fprint(w, code_BadPOSTdata) // Error in Data
		return
	}

	// Message's Length ?
	if len(p2) > msgMaxSize {
		log.Printf("Too long Message, %d Bytes.", len(p2))
		fmt.Fprint(w, code_msgTooLong) // Too long Message
		return
	}

	// Size Match
	if utf8.RuneCountInString(p2) != int(p1_int64) {
		log.Print("Size does not match.")
		fmt.Fprint(w, code_BadPOSTdata) // The Size does not match the Contents!
		return
	}

	// HTML safe Text
	txt_safe = html.EscapeString(p2)

	// Create Job for ChatManager
	rcvChan = make(chan tChatJob)
	chatJob = new(tChatJob)
	chatJob.chatRecord.author = uid
	chatJob.chatRecord.message = txt_safe
	chatJob.returnChannel = rcvChan

	// Send Job
	chatManagerChan <- *chatJob

	// Wait for Manager
	*chatJob = <-rcvChan

	fmt.Fprint(w, code_messageSent) // OK, Message is Sent

}

//------------------------------------------------------------------------------

func page_activeList(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request of Active Users List Page.
	// Gives Client a List of active Clients.

	// Client sends an empty GET Request.

	// Server replies to client one of the following:
	//		1. code_NotLoggedIn ('L'),
	//		2. JSON (list_of_active_clients).

	var ok bool
	var rcvChan chan tActiveJob
	var activeJob *tActiveJob

	// Correct Cookies & Not Idle ?  & update User's Last Activity Time
	ok, _, _, _ = user_check(w, req)
	// Also updates User's Last Activity Time

	// Logged in ?
	if !ok {
		fmt.Fprint(w, code_NotLoggedIn) // Not Logged In
		return
	}

	// Requesting the Manager
	// Create Job
	rcvChan = make(chan tActiveJob)
	activeJob = new(tActiveJob)
	activeJob.action = activeJobGetList // Get List
	activeJob.returnChannel = rcvChan

	// Send Job
	activeManagerChan <- *activeJob

	// Get Feedback
	*activeJob = <-rcvChan

	// Send to Client
	fmt.Fprint(w, activeJob.client.address) // List was packed into "address"

}

//------------------------------------------------------------------------------

func page_index(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request of Index Page.

	var ok bool

	// Correct Cookies & Not Idle ?  & update User's Last Activity Time
	ok, _, _, _ = user_check(w, req)

	if ok {
		// User is logged-in (cookie matches) & active
		fmt.Fprintf(w, "%sYou are already logged in.<br>This page refreshes in %s seconds.<br>Click <a href='%s'>here</a> to proceed, if your web browser does not support redirects.%s",
			html_1_toChat, redirectDelay_str, path_chat, html_2) //
		return
	}

	fmt.Fprint(w, tpl_index)
}

//------------------------------------------------------------------------------

func page_chat(w http.ResponseWriter, req *http.Request) {

	// Process and serve User's Request of Chat Page.

	var ok bool

	// Correct Cookies & Not Idle ?  & update User's Last Activity Time
	ok, _, _, _ = user_check(w, req)

	if !ok {
		fmt.Fprintf(w, "%sCan not enter the Chat.<br>If your previous Session has not been properly closed, then, please, wait for it to be automatically terminated.<br>Click <a href='%s'>here</a> to return to index Page.%s",
			html_1, path_index, html_2) //
		return
	}

	fmt.Fprint(w, tpl_chat)

}

//------------------------------------------------------------------------------

func page_stat(w http.ResponseWriter, req *http.Request) {

	// Statistics of the Server.

	fmt.Fprint(w, tpl_registeredUsersList) //
}

//------------------------------------------------------------------------------

func page_login(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Log-In Request
	// (made from Index or other Page).

	var cookie_1 http.Cookie
	var cookie_2 http.Cookie
	var err error
	var uid_str, pwd, qid_str, qa_str, sid_str, sid_b64 string
	var uid, qid, qa_uint64 uint64
	var qa, correctAnswer uint8
	var exists bool
	var rcvChan chan tAsqJob
	var rcv2Chan chan tLoginJob
	var asqJob *tAsqJob
	var loginJob *tLoginJob
	var delay int64
	var ok bool
	var sid uint32

	// Parse Form
	err = req.ParseForm()
	if err != nil {
		log.Println("Error Reading POST Form:", err) //
		fmt.Fprintf(w, "%sBad POST Data. If this error repeats, contact the administrator of this chat.%s",
			html_1, html_2) //
		return
	}

	// Read Parameters
	uid_str = req.PostFormValue(param_login_userID)
	pwd = req.PostFormValue(param_login_password)
	qid_str = req.PostFormValue(param_qid)
	qa_str = req.PostFormValue(param_qAnswer)

	// Check UID
	uid, err = strconv.ParseUint(uid_str, 10, 64)
	if err != nil {
		log.Println("Bad UID:", err) //
		fmt.Fprintf(w, "%sError. UID is bad.<br>Click <a href='%s'>here</a> to return to main page. %s",
			html_1, path_index, html_2) //
		return
	}

	// Check QID
	qid, err = strconv.ParseUint(qid_str, 10, 64)
	if err != nil {
		log.Println("Error in QID.", err) //
		fmt.Fprintf(w, "%sLogging failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Check QA
	qa_uint64, err = strconv.ParseUint(qa_str, 10, 64)
	if err != nil {
		log.Println("Error in QAnswer.", err) //
		fmt.Fprintf(w, "%sLogging failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}
	qa = uint8(qa_uint64)

	// Check Anti-Spam Answer
	_, exists = asqsList[qid]
	if !exists {
		log.Println("UnExisting QID!") //
		fmt.Fprintf(w, "%sLogging failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Get ASQ by QID
	// Create Job
	rcvChan = make(chan tAsqJob)
	asqJob = new(tAsqJob)
	asqJob.action = asqJobGet // Get
	asqJob.qid = qid
	asqJob.returnChannel = rcvChan

	// Send Job
	asqManagerChan <- *asqJob

	// Wait for Feedback
	*asqJob = <-rcvChan

	correctAnswer = asqJob.asq.answer // instead of thread-unsafe: asqsList[qid].answer
	if qa != correctAnswer {
		fmt.Fprintf(w, "%sLogging failed.<br>The Answer to anti-spam Question is wrong.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Check Question Timeout
	delay = time.Now().Unix() - asqJob.asq.timeOfCreation // instead of thread-unsafe: asqsList[qid].timeOfCreation
	if delay > asqTimeout {
		fmt.Fprintf(w, "%sLogging failed.<br>The Answer is outdated.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Session Exists ?
	_, exists = activeClientsList[uid]
	if exists {
		// Already Logged In!
		fmt.Fprintf(w, "%sLogging failed.<br>Already logged in!<br>Click <a href='%s'>here</a> to return to Chat page. %s",
			html_1_toChat, path_chat, html_2) //
		return
	}

	// Check UID:PWD Combination
	ok = user_isGood(uid, &pwd) // User exists & Passowrd is correct
	if !ok {
		fmt.Fprintf(w, "%sLogging failed.<br>Bad UID or Password.<br>Click <a href='%s'>here</a> to return to main page. %s",
			html_1, path_index, html_2) //
		return
	}

	// Create SID
	// SID may be not unique, because Key in activeClients Map is UID, not SID.
	sid = generateRandomUint32()
	sid_str = fmt.Sprintf("%d", sid)
	sid_b64 = base64.StdEncoding.EncodeToString([]byte(sid_str))

	// Create LoginJob
	rcv2Chan = make(chan tLoginJob)
	loginJob = new(tLoginJob)
	loginJob.returnChannel = rcv2Chan
	loginJob.client.address = req.RemoteAddr
	loginJob.client.sid = sid_b64
	loginJob.uid = uid

	// Send LoginJob
	loginManagerChan <- *loginJob

	// Get Feedback
	*loginJob = <-rcv2Chan

	if loginJob.result != true {
		// Already Logged In!
		fmt.Fprintf(w, "%sLogging failed.<br>Already logged in!<br>Click <a href='%s'>here</a> to return to Chat page. %s",
			html_1_toChat, path_chat, html_2) //
		return
	}

	// Set Session cookie
	cookie_1.HttpOnly = true
	cookie_1.Secure = false
	cookie_1.Name = "SID"
	cookie_1.Value = sid_b64
	http.SetCookie(w, &cookie_1)

	cookie_2.HttpOnly = true
	cookie_2.Secure = false
	cookie_2.Name = "UID"
	cookie_2.Value = fmt.Sprintf("%d", uid)
	http.SetCookie(w, &cookie_2)

	// HTML
	fmt.Fprintf(w, "%sYou are now logged in.<br>This page refreshes in %s seconds.<br>Click <a href='%s'>here</a> to proceed, if your web browser does not support redirects.%s",
		html_1_toChat, redirectDelay_str, path_chat, html_2) //
}

//------------------------------------------------------------------------------

func page_logout(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Log-Out Request
	// (made from Index or other Page).
	// Checks if User has correct Cookies and is Not Idle, then Logs Out.

	var cookie_1, cookie_2 *http.Cookie
	var err_1, err_2, err_3 error
	var client_uid, client_sid string
	var uid uint64
	var exists bool
	var rcvChan chan tActiveJob
	var activeJob *tActiveJob

	// Read Client's Cookies
	cookie_1, err_1 = req.Cookie("UID")
	cookie_2, err_2 = req.Cookie("SID")
	if (err_1 != nil) || (err_2 != nil) {
		// No Cookies
		return
	}

	// Checking Cookies
	client_uid = cookie_1.Value
	client_sid = cookie_2.Value
	uid, err_3 = strconv.ParseUint(client_uid, 10, 64)
	if err_3 != nil {
		log.Println("Bad UID in Cookie:", err_3) //
		return
	}

	// Active Client exists ?
	_, exists = activeClientsList[uid]
	if exists {
		// SID match ?

		// Get SID by UID
		// Create Job
		rcvChan = make(chan tActiveJob)
		activeJob = new(tActiveJob)
		activeJob.action = activeJobGetUser // Get User (for SID)
		activeJob.uid = uid
		activeJob.returnChannel = rcvChan

		// Send Job
		activeManagerChan <- *activeJob

		// Get Feedback
		*activeJob = <-rcvChan

		if client_sid == activeJob.client.sid { // instead of thread-unsafe: activeClientsList[uid].sid

			// SID matches
			// Delete from Active List

			// Modify previous Job
			activeJob.action = activeJobDelete // Delete

			// Send Job
			activeManagerChan <- *activeJob

			// Get Feedback
			*activeJob = <-rcvChan // Result is not needed

			// Now ask activeManager to resync List of active Users

			// Modify previous Job
			activeJob.action = activeJobUpdateCache // Update Cache
			activeManagerChan <- *activeJob
			*activeJob = <-rcvChan

		}
	}

	// Delete Cookies
	// Set Session cookie
	cookie_1.HttpOnly = true
	cookie_1.Secure = false
	cookie_1.Name = "SID"
	cookie_1.Value = ""
	cookie_1.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie_1)

	cookie_2.HttpOnly = true
	cookie_2.Secure = false
	cookie_2.Name = "UID"
	cookie_2.Value = ""
	cookie_2.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie_2)

	fmt.Fprintf(w, "%sYou are logged off.<br>This page refreshes in %s seconds.<br>Click <a href='%s'>here</a> to proceed, if your web browser does not support redirects.%s",
		html_1_toIndex, redirectDelay_str, path_index, html_2) //
}

//------------------------------------------------------------------------------

func page_register(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request of Registration Request
	// (made from Index or other Page).

	var ok bool = true
	var exists bool
	var uid, qid, qa_uint64 uint64
	var err error
	var userName, pwd, qid_str, qa_str string
	var qa, correctAnswer uint8
	var rcvChan chan tAsqJob
	var rcv2Chan chan tRegisterJob
	var asqJob *tAsqJob
	var regJob *tRegisterJob
	var delay int64

	// Parse Form
	err = req.ParseForm()
	if err != nil {
		log.Println("Error Reading POST Form:", err) //
		fmt.Fprintf(w, "%sRegistration failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Read Parameters
	userName = req.PostFormValue(param_reg_userName)
	pwd = req.PostFormValue(param_reg_password)
	qid_str = req.PostFormValue(param_qid)
	qa_str = req.PostFormValue(param_qAnswer)

	if (len(userName) > userName_maxLen) || (len(pwd) > userPwd_maxLen) {
		log.Println("Too long Name or Password") //
		fmt.Fprintf(w, "%sRegistration failed.<br>Name or Password is too long.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Check QID
	qid, err = strconv.ParseUint(qid_str, 10, 64)
	if err != nil {
		log.Println("Error in QID. qid=[", qid_str, "].", err) //
		fmt.Fprintf(w, "%sRegistration failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Check QA
	qa_uint64, err = strconv.ParseUint(qa_str, 10, 64)
	if err != nil {
		log.Println("Error in QAnswer.", err) //
		fmt.Fprintf(w, "%sRegistration failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}
	qa = uint8(qa_uint64)

	// Check Anti-Spam Answer

	// ASQ exists ?
	_, exists = asqsList[qid]
	if !exists {
		log.Println("UnExisting QID!") //
		fmt.Fprintf(w, "%sRegistration failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Create Job
	rcvChan = make(chan tAsqJob)
	asqJob = new(tAsqJob)
	asqJob.action = asqJobGet // Get
	asqJob.qid = qid
	asqJob.returnChannel = rcvChan

	// Send Job
	asqManagerChan <- *asqJob

	// Wait for Feedback
	*asqJob = <-rcvChan

	correctAnswer = asqJob.asq.answer // instead of thread-unsafe: asqsList[qid].answer
	if qa != correctAnswer {
		fmt.Fprintf(w, "%sRegistration failed.<br>The Answer to anti-spam Question is wrong.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Check Question Timeout
	delay = time.Now().Unix() - asqJob.asq.timeOfCreation // instead of thread-unsafe: asqsList[qid].timeOfCreation
	if delay > asqTimeout {
		fmt.Fprintf(w, "%sRegistration failed.<br>The Answer is outdated.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
		return
	}

	// Register

	// Create Job
	rcv2Chan = make(chan tRegisterJob)
	regJob = new(tRegisterJob)
	regJob.name = userName
	regJob.pwd = pwd
	regJob.returnChannel = rcv2Chan

	// Send Job
	registerManagerChan <- *regJob

	// Wait for Feedback
	*regJob = <-rcv2Chan
	ok = regJob.result
	uid = regJob.uid

	if !ok {
		fmt.Fprintf(w, "%sRegistration failed.<br>Click <a href='%s'>here</a> to return to main Page.%s",
			html_1, path_index, html_2) //
	}

	// Saves all the registered Users to a string
	saveRegisteredUsersToTpl()

	// Reply to the Client
	fmt.Fprint(w, tpl_userRegistered_p1,
		fmt.Sprintf(tpl_userRegistered_p2, uid),
		tpl_userRegistered_p3) //
}

//------------------------------------------------------------------------------

func page_asq(w http.ResponseWriter, req *http.Request) {

	// Processes and serves User's Request of Anti-Spam Question.

	var qid uint64
	var rcvChan chan tAsqJob
	var asqJob *tAsqJob
	var msg string

	// Client sends an empty GET Request.

	// Server replies to client following:
	//		1. JSON (question).

	qid = asq_create() // QID given by asqManager is unique

	// Create Job
	rcvChan = make(chan tAsqJob)
	asqJob = new(tAsqJob)
	asqJob.action = asqJobGet // Get
	asqJob.qid = qid
	asqJob.returnChannel = rcvChan

	// Send Job
	asqManagerChan <- *asqJob

	// Wait for Feedback
	*asqJob = <-rcvChan

	msg = base64.StdEncoding.EncodeToString(asqJob.asq.question)
	// instead of thread-unsafe: asqsList[qid].question

	// Server's Reply in JSON Format
	// {"qid":"123","msg":"AbRaKaDaBrA="}
	fmt.Fprint(w, "{\"qid\":\"", qid, "\",\"msg\":\"", msg, "\"}")

	// Clear Question Data from ASQ
	asq_clearQuestionData(qid)
}

//------------------------------------------------------------------------------
