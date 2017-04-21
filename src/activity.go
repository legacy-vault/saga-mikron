// activity.go

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"time"
)

//------------------------------------------------------------------------------

// Lists
type tActiveClient struct {
	log_mid uint16 // ID of the last "actual" message in Chat before Client's
	// Log-In. If there are no actual Messages (as at fresh Start), then it is
	// just a Counter which shows where to start from. Client will not see this
	// last Message (whether it is actual or not), he sees only next Messages.
	log_ts         int64  // Timestamp of the Log-In Event
	sid            string // Session ID of a Client
	address        string // Address of a Client
	lastActiveTime int64  // Time of last Activity of a Client
}
type tActiveClients map[uint64]tActiveClient // Key = UID

type tActiveJob struct {
	uid           uint64
	client        tActiveClient
	returnChannel chan tActiveJob
	action        uint8
}

//------------------------------------------------------------------------------

const activityRevisorInterval_default = 30 // Interval for Monitoring Active Clients, in Seconds
const idleTimeout = 120                    // Idle Client Timeout, in Seconds
const activeManagerChanBufferLen = 64      // Buffer Length of the Active Manager's Channel

const activeJobDelete = 1      // Action Code for Active Manager to Delete User from List
const activeJobUpdateUser = 2  // Action Code for Active Manager to Update User's L.A.T.
const activeJobGetUser = 3     // Action Code for Active Manager to Get User's Information
const activeJobGetList = 4     // Action Code for Active Manager to Get List of active Clients
const activeJobUpdateCache = 5 // Action Code for Active Manager to update the cached List (in JSON Format)
//const activeJobResetLMID = 6   // Action Code for Active Manager to Reset All-User's LMID

//------------------------------------------------------------------------------

// Internal Parameters
var activityRevisorInterval int

// Lists
var activeClientsList tActiveClients

// Channels
var activeRevisorQuit chan int
var activeManagerQuit chan int
var activeManagerChan chan tActiveJob

//------------------------------------------------------------------------------

func activeRevisor() {

	// Activity Revisor periodically checks all active Users whether they are
	// really active.

	var lat, now, criterion int64 // Last Activity Time of a User
	var loop bool = true
	var rcvChan chan tActiveJob
	var i, count uint64
	var v tActiveClient
	var activeJob *tActiveJob

	// Preparations
	rcvChan = make(chan tActiveJob)
	activeJob = new(tActiveJob)
	activeJob.action = activeJobDelete // Delete
	activeJob.returnChannel = rcvChan

	// Periodical Check for active Clients
	for loop {

		// Starting Job
		now = time.Now().Unix()
		criterion = now - idleTimeout
		count = 0 // Count of deleted Users

		// For each active Client
		for i, v = range activeClientsList {

			lat = v.lastActiveTime
			if lat < criterion {
				// Idle User must be deleted from the List
				count++

				// Re-Configure Job for specific idle Client
				activeJob.uid = i

				// Send Job
				activeManagerChan <- *activeJob

				// Get Feedback
				*activeJob = <-rcvChan
			}

		}

		// After all idle Clients are deleted from active Clients's List,
		// Ask activeManager to resync List of active Users
		if count > 0 {

			// Modify previous Job
			activeJob.action = activeJobUpdateCache // Update Cache
			activeJob.uid = 0
			activeManagerChan <- *activeJob
			*activeJob = <-rcvChan
		}

		// Checking for Stop Signal
		select {
		case <-activeRevisorQuit:
			loop = false
			log.Println("Closing Activity Revisor...") //
		default:

		}

		// Wait for next Job
		time.Sleep(time.Second * time.Duration(activityRevisorInterval))
	}
}

//------------------------------------------------------------------------------

func activeManager() {

	// Manages active Clients.

	var loop bool = true
	var job tActiveJob
	var tmp_client tActiveClient
	var key uint64
	var activeUsersListJSON, text string // A cached List of active Clients
	var buffer bytes.Buffer
	var count, cur int

	// Preparations
	activeUsersListJSON = "{\"names\":[]}" // Initial is empty, Server has just started.

	for loop {

		job = <-activeManagerChan // Get Job from Channel

		if job.action == activeJobUpdateUser { // Update

			tmp_client = activeClientsList[job.uid]
			tmp_client.lastActiveTime = time.Now().Unix()
			activeClientsList[job.uid] = tmp_client

		} else if job.action == activeJobGetList { // Get List

			// Pack List into "address" field, as it is the same string
			job.client.address = activeUsersListJSON

		} else if job.action == activeJobGetUser { // Get User

			job.client = activeClientsList[job.uid]

		} else if job.action == activeJobDelete { // Delete

			delete(activeClientsList, job.uid)

		} else if job.action == activeJobUpdateCache { // Update Cache

			// Output in JSON Format
			buffer.WriteString("{\"names\":[")
			count = len(activeClientsList)
			cur = 1
			for key, _ = range activeClientsList {
				text = base64.StdEncoding.EncodeToString([]byte(userDataList[key].name))
				if cur < count {
					// not last
					buffer.WriteString(fmt.Sprintf("\"%s\",", text))
				} else {
					// last element
					buffer.WriteString(fmt.Sprintf("\"%s\"", text))
				}
				cur++
			}
			buffer.WriteString("]}")
			activeUsersListJSON = buffer.String()
			buffer.Reset()

		}

		// Feedback
		job.returnChannel <- job

		// Checking for Stop Signal
		select {
		case <-activeManagerQuit:
			loop = false
			log.Println("Closing Active Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------
