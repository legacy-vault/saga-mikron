// user_data.go

package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

//------------------------------------------------------------------------------

// Lists
type tUserData struct {
	name     string // User's Name
	pwd      string // User's Password
	reg_time int64  // Time of Registration, Unix Timestamp
}
type tUserDatas map[uint64]tUserData // Key = UID

//------------------------------------------------------------------------------

const file_userData_default = "dat/user.dat" // Path to File with User Data

//------------------------------------------------------------------------------

// Lists
var userDataList tUserDatas

// File
var file_userData string
var createUserDataFile bool // Should we create User Data File if it does not exist ?

//------------------------------------------------------------------------------

func userData_init() (ok bool) {

	// Reads the User-Data File (U.D.F.) into Memory.
	// If the U.D.F. does not exist (e.g. at first Start), then a new U.D.F.
	// is created.

	var err error
	var ptr *tUserDatas

	_, err = os.Stat(file_userData)
	if err != nil {
		if os.IsNotExist(err) {

			// File not found, Create a new one ?
			if createUserDataFile {
				userData_create(&file_userData)
			} else {

				log.Println("User Data File not found.", file_userData) //
				return false
			}

		} else {

			// Other Error
			log.Println("Error with File Stat", file_userData) //
			return false
		}
	}

	// File exists
	ptr = userData_read(file_userData)
	if ptr == nil {

		log.Println("Error getting userData") //
		return false
	}

	userDataList = *ptr
	return true
}

//------------------------------------------------------------------------------

func userData_create(fileName *string) {

	// Creates a User-Data File.
	// Notes:
	// If the file already exists, the function will not create a new file, id
	// est, this function should be run only if you are sure that file does
	// not exist.

	var file *os.File
	var err error

	// Create a new file if none exists
	file, err = os.OpenFile(*fileName, os.O_CREATE, 0755)
	if err != nil {
		log.Println("Error creating config at", fileName, err) //
		return
	}

	// Close File immediately
	err = file.Close()
	if err != nil {
		log.Println("Error closing file", fileName, err) //
		return
	}

	// Create a system User
	userData_creatSysUser(fileName)
}

//------------------------------------------------------------------------------

func userData_read(fileName string) (ud *tUserDatas) {

	// Reads User Data.
	// If Errors occur, the function returns nil Pointer.

	var file *os.File
	var err error
	var reader *bufio.Reader
	var t1buf, t2buf, t3buf, t4buf, t5buf, t6buf []byte
	var n int
	var t1 uint64
	var t2 int64
	var t3, t5 uint8
	var t4, t6 string
	var udt tUserDatas
	var userData *tUserData

	file, err = os.Open(fileName)
	if err != nil {
		log.Println("Error opening user data file at", fileName, err) //
		return nil
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println("Error closing file", fileName, err) //
		}
	}()

	reader = bufio.NewReader(file)
	t1buf = make([]byte, 8)
	t2buf = make([]byte, 8)
	t3buf = make([]byte, 1)
	t4buf = []byte{}
	t5buf = make([]byte, 1)
	t6buf = []byte{}

	udt = make(tUserDatas)

	for {
		// Read UID [8 Bytes]
		n, err = reader.Read(t1buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < 8 {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t1 = binary.LittleEndian.Uint64(t1buf)

		// Read RegTime [8 Bytes]
		n, err = reader.Read(t2buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < 8 {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t2 = int64(binary.LittleEndian.Uint64(t2buf))

		// Read Length of PWD [1 Byte]
		n, err = reader.Read(t3buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < 1 {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t3 = t3buf[0]

		// Read PWD [Several Bytes]
		t4buf = make([]byte, t3)
		n, err = reader.Read(t4buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < int(t3) {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t4 = string(t4buf)

		// Read Length of Name [1 Byte]
		n, err = reader.Read(t5buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < 1 {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t5 = t5buf[0]

		// Read Name [Several Bytes]
		t6buf = make([]byte, t5)
		n, err = reader.Read(t6buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from file.", err) //
				return nil
			} else {
				break
			}
		}

		if n < int(t5) {
			log.Println("Read Portion is Less than it should be!") //
			return nil
		}
		t6 = string(t6buf)

		// Filling ud
		userData = new(tUserData)
		userData.name = t6
		userData.pwd = t4
		userData.reg_time = t2
		udt[t1] = *userData
	}

	ud = &udt
	return ud

}

//------------------------------------------------------------------------------

func userData_creatSysUser(fileName *string) {

	// Adds a system User to the existing User-Data File (U.D.F.).
	// Used only when no U.D.F. exists.

	// Create User in Memory
	var pwd string
	var i, x, rnd_len uint8
	var buf []byte
	var ok bool
	var ud *tUserData
	var file *os.File
	var err error

	// Prepare Data
	rnd_len = uint8(rand.Uint32())
	if rnd_len < 127 {
		rnd_len += 128
	}
	buf = make([]byte, rnd_len)
	for i = 0; i < rnd_len; i++ {
		x = uint8(rand.Uint32())
		buf[i] = x
	}
	pwd = string(buf)

	// Create Struct
	ud = new(tUserData)
	ud.name = chat_systemUserName
	ud.reg_time = time.Now().Unix()
	ud.pwd = pwd

	// Write User to the File, without adding to the List
	// The List does not yet exist.

	// Open the File
	file, err = os.OpenFile(*fileName, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		log.Println("Error opening config at", file_userData, err) //
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println("Error closing file", file_userData, err) //
		}
	}()

	ok = userData_write(ud, chat_systemUserUID, file)
	if !ok {
		log.Println("Error during writing user to file") //
		return
	}
}

//------------------------------------------------------------------------------

func userData_write(ud *tUserData, uid uint64, file io.Writer) (ok bool) {

	// Outputs the given User Data to the User-Data File.

	var t1 uint64 = uid
	var t2 int64 = ud.reg_time
	var t3, t5 uint8  // Length of ud.pwd & ud.name
	var t4, t6 []byte // ud.pwd & ud.name
	var err error
	var t4_len, t6_len int

	t4 = []byte(ud.pwd)
	t4_len = len(t4)
	if t4_len <= 255 {
		t3 = uint8(t4_len)
	} else {
		log.Println("Too long pwd") //
		return false
	}

	t6 = []byte(ud.name)
	t6_len = len(t6)
	if t6_len <= 255 {
		t5 = uint8(t6_len)
	} else {
		log.Println("Too long Name") //
		return false
	}

	// Output to File
	err = binary.Write(file, binary.LittleEndian, t1)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	err = binary.Write(file, binary.LittleEndian, t2)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	err = binary.Write(file, binary.LittleEndian, t3)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	err = binary.Write(file, binary.LittleEndian, t4)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	err = binary.Write(file, binary.LittleEndian, t5)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	err = binary.Write(file, binary.LittleEndian, t6)
	if err != nil {
		log.Println("Error printing to file", file_userData, err) //
		return false
	}
	return true
}

//------------------------------------------------------------------------------

func user_register(name, pwd *string) (ok bool, uid uint64) {

	// Generates a random UserID,
	// Creates a new User with given Name, Password & created UserID,
	// Writes all this to the User-Data File and into Memory.

	ok = true
	var tmp_uid uint64
	var exists bool = true
	var ud *tUserData
	var err error
	var file *os.File

	if (len(*name) > userName_maxLen) || (len(*pwd) > userPwd_maxLen) {
		log.Println("Too long Name or Password") //
		return false, 0
	}

	// Generating random UID
	for exists {
		tmp_uid = rand.Uint64()
		_, exists = userDataList[tmp_uid]
		if !exists {
			// found free UID
			exists = false
			break
		}
	}

	// Create a Struct and Add to the List
	ud = new(tUserData)
	ud.name = *name
	ud.pwd = *pwd
	ud.reg_time = time.Now().Unix()
	userDataList[tmp_uid] = *ud

	// Adding to File
	file, err = os.OpenFile(file_userData, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		log.Println("Error opening config at", file_userData, err) //
		return false, 0
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println("Error closing file", file_userData, err) //
		}
	}()

	ok = userData_write(ud, tmp_uid, file)
	return ok, tmp_uid
}

//------------------------------------------------------------------------------

func user_isGood(uid uint64, pwd *string) (ok bool) {

	// Checks if User exists and User's Password matches the Actual Password.

	var ud tUserData
	var exists bool

	ud, exists = userDataList[uid]
	if exists {
		if ud.pwd == *pwd {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

//------------------------------------------------------------------------------

func user_check(w http.ResponseWriter, req *http.Request) (cookies_ok bool, user_uid uint64, user_lmid uint16, user_logts int64) {

	// Checks if User has correct Cookies and is Not Idle.

	var log_mid uint16
	var uid uint64
	var log_ts, timeInactive int64
	var err_1, err_2, err_3 error
	var cookie_uid, cookie_sid *http.Cookie
	var cookie_uid_str, cookie_sid_str string
	var rcvChan chan tActiveJob
	var activeJob *tActiveJob
	var exists bool

	// Read Client's Cookies
	cookie_uid, err_1 = req.Cookie("UID")
	cookie_sid, err_2 = req.Cookie("SID")
	if (err_1 != nil) || (err_2 != nil) {
		//log.Println("No Cookies:", err_1, err_2) //dbg
		// No Cookies
		return false, 0, 0, 0
	}

	// Cookie -> string & Parse UID
	cookie_uid_str = cookie_uid.Value
	cookie_sid_str = cookie_sid.Value
	uid, err_3 = strconv.ParseUint(cookie_uid_str, 10, 64)
	if err_3 != nil {
		log.Println("user_check: Bad UID in Cookie:", err_3) //dbg
		return false, 0, 0, 0
	}

	// Active Client Existance
	_, exists = activeClientsList[uid]
	if !exists {
		//log.Println("User is not active") //dbg
		return false, 0, 0, 0
	}

	// Get Information about Client

	// Create Job
	rcvChan = make(chan tActiveJob)
	activeJob = new(tActiveJob)
	activeJob.action = activeJobGetUser // Get User
	activeJob.returnChannel = rcvChan
	activeJob.uid = uid
	//
	// Send Job
	activeManagerChan <- *activeJob
	//
	// Get Feedback
	*activeJob = <-rcvChan

	// SID Match
	if cookie_sid_str != activeJob.client.sid { // instead of thread-unsafe: activeClientsList[uid].sid
		//log.Println("SID does not match. Cookie has", cookie_sid_str, ", needed", activeJob.client.sid) //dbg
		return false, 0, 0, 0
	}

	// Really active or Revisor is sleeping ?
	timeInactive = time.Now().Unix() - activeJob.client.lastActiveTime // instead of thread-unsafe: activeClientsList[uid].lastActiveTime
	if timeInactive <= idleTimeout {

		// User is active
		// Update last Activity Time -> activeManager

		// Modify Job
		activeJob.action = activeJobUpdateUser // Update L.A.T.

		// Send Job
		activeManagerChan <- *activeJob

		// Get Feedback
		*activeJob = <-rcvChan

		log_mid = activeJob.client.log_mid // instead of thread-unsafe: lmid = activeClientsList[uid].log_mid
		log_ts = activeJob.client.log_ts

		return true, uid, log_mid, log_ts

	} else {

		// User is idle
		//log.Println("User is idle") //dbg
		// Delete from Active List -> activeManager

		// Create Job
		rcvChan = make(chan tActiveJob)
		activeJob = new(tActiveJob)
		activeJob.action = activeJobDelete // Delete
		activeJob.returnChannel = rcvChan
		activeJob.uid = uid

		// Send Job
		activeManagerChan <- *activeJob

		// Get Feedback
		*activeJob = <-rcvChan

		// Session has ended
		return false, 0, 0, 0
	}
}

//------------------------------------------------------------------------------
