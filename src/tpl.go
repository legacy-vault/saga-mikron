// tpl.go

package main

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"strings"
)

//------------------------------------------------------------------------------

// Configuration-Template Files
const file_index_default = "tpl/index.html"                    // Path to Index Page Template
const file_chat_default = "tpl/chat.html"                      // Path to Chat Page Template
const file_userRegistered_default = "tpl/user_registered.html" // Path to 'User Registered' Page Template
const tpl_sep = "//#//"                                        // Separator of variable Part

// These are HTML-Parts for "small" Pages (Redirectors or Errors).
const html_headTitle = "Chat" // Contents of the <head><title>...</title></head>
const html_tdTitle = "Chat"   // Contents of the Upper-Left Corner Cell on Pages
const html_1 = "<html><head><meta charset='utf-8'><title>" + html_headTitle + "</title></head>\n<body>\n"
const html_1_toChat = "<html><head><meta charset='utf-8'><title>" + html_headTitle + "</title>" +
	"<meta http-equiv='refresh' content='" + redirectDelay_str + "; url=" + path_chat + "'/></head>\n<body>\n"
const html_1_toIndex = "<html><head><meta charset='utf-8'><title>" + html_headTitle + "</title>" +
	"<meta http-equiv='refresh' content='" + redirectDelay_str + "; url=" + path_index + "'/></head>\n<body>\n"
const html_2 = "</body></html>"

//------------------------------------------------------------------------------

// Contents of a File, Template
var tpl_index, tpl_registeredUsersList string
var tpl_chat string
var tpl_userRegistered_p1, tpl_userRegistered_p2, tpl_userRegistered_p3 string // 3 Parts

// Internal Parameters
var tpl_sep_len int

// Path to File
var file_indexTemplate, file_chatTemplate, file_userRegdTemplate string

//------------------------------------------------------------------------------

func templates_init() (ok bool) {

	// Reads the Templates of Pages from Files into Memory.

	ok = false

	tpl_sep_len = len(tpl_sep)

	ok = template_index()
	if !ok {
		return ok
	}

	ok = template_chat()
	if !ok {
		return ok
	}

	ok = template_userRegistered()
	if !ok {
		return ok
	}

	return ok
}

//------------------------------------------------------------------------------

func template_index() (ok bool) {

	// Prepares the Template of Index Page.

	var buffer []byte
	var err error
	var tpl_tmp, tpl_part1_tmp, tpl_part_1, tpl_part_2 string
	var tpl_sep_pos int

	// File -> []byte -> string
	buffer, err = ioutil.ReadFile(file_indexTemplate)
	if err != nil {
		log.Println("Error reading file", file_indexTemplate, err) //
		return false
	}
	tpl_tmp = string(buffer) // Buffer -> string

	// Find Sepatator
	tpl_sep_pos = strings.Index(tpl_tmp, tpl_sep)

	// Split first Part from string and fill it
	tpl_part1_tmp = tpl_tmp[:tpl_sep_pos]
	tpl_part_1 = fmt.Sprintf(tpl_part1_tmp,
		html_headTitle,
		html_tdTitle,
		param_login_userID,
		param_login_password,
		param_reg_userName,
		param_reg_password,
		param_qid,
		param_qAnswer,
		path_login,
		path_register,
		path_stat,
		path_asq,
		srv_protocol)

	// Split second Part
	tpl_part_2 = tpl_tmp[tpl_sep_pos:]

	// Join Parts
	tpl_index = tpl_part_1 + tpl_part_2

	return true
}

//------------------------------------------------------------------------------

func template_chat() (ok bool) {

	// Prepares the Template of Chat Page.

	var buffer []byte
	var err error
	var tpl_tmp, tpl_part1_tmp, tpl_part_1, tpl_part_2 string
	var tpl_sep_pos int

	// File -> Buffer -> string
	buffer, err = ioutil.ReadFile(file_chatTemplate)
	if err != nil {
		log.Println("Error reading file", file_chatTemplate, err) //
		return false
	}
	tpl_tmp = string(buffer) // Buffer -> string

	// Find Sepatator
	tpl_sep_pos = strings.Index(tpl_tmp, tpl_sep)

	// Split first Part from string and fill it
	tpl_part1_tmp = tpl_tmp[:tpl_sep_pos]
	tpl_part_1 = fmt.Sprintf(tpl_part1_tmp,
		html_headTitle,
		html_tdTitle,
		path_index,
		path_news,
		path_send,
		path_activeList,
		path_logout,
		srv_protocol,
		code_NoNews,
		code_BadPOSTdata,
		code_BadRequest,
		code_NotLoggedIn,
		code_EmptyMessage,
		code_messageSent,
		code_msgTooLong,
		redirectDelay_str,
		sendToGetDelay_str,
		msgUpdateInterval_str,
		userUpdateInterval_str,
		msgMaxSize,
		param_req_mid,
		param_req_ts,
		param_unknownVal)

	// Split second Part
	tpl_part_2 = tpl_tmp[tpl_sep_pos:]

	// Join strings
	tpl_chat = tpl_part_1 + tpl_part_2

	return true
}

//------------------------------------------------------------------------------

func template_userRegistered() (ok bool) {

	// Prepares the Template of 'User Registered' Page.

	var buffer []byte
	var err error
	var tpl_tmp, tpl_part1_tmp string
	var tpl_sep1_pos, tpl_sep2_pos int

	// File -> Buffer -> string
	buffer, err = ioutil.ReadFile(file_userRegdTemplate)
	if err != nil {
		log.Println("Error reading file", file_userRegdTemplate, err) //
		return false
	}
	tpl_tmp = string(buffer) // Buffer -> string

	// Since there is no built-in 'charAt' function for UTF-8 strings in Go,
	// we use owr own ideas. We do not use the built-in 'strings.Index'
	// function while we need to find more than one strings and do not want to
	// search the same string several times.

	// Find First Sepatator
	tpl_sep1_pos = strings.Index(tpl_tmp, tpl_sep)

	// Split first Part from string and fill it
	tpl_part1_tmp = tpl_tmp[:tpl_sep1_pos]
	tpl_userRegistered_p1 = fmt.Sprintf(tpl_part1_tmp,
		html_headTitle,
		srv_protocol,
		path_index,
		html_tdTitle)

	// Find Second Sepatator
	tpl_sep2_pos = strings.Index(tpl_tmp[tpl_sep1_pos+tpl_sep_len:], tpl_sep)

	// Split second Part, without filling
	tpl_userRegistered_p2 = tpl_tmp[tpl_sep1_pos : tpl_sep1_pos+tpl_sep_len+tpl_sep2_pos]
	// tpl_userRegistered_p2 will be filled during each User's Registration

	// Split third Part
	tpl_userRegistered_p3 = tpl_tmp[tpl_sep1_pos+tpl_sep_len+tpl_sep2_pos:]

	return true
}

//------------------------------------------------------------------------------

func saveRegisteredUsersToTpl() {

	// Saves all the registered Users to a string.
	// ! Must be run after userData_init() !

	var buffer *bytes.Buffer
	var i uint64
	var v tUserData

	// Save Template
	buffer = bytes.NewBuffer(nil)
	buffer.WriteString(html_1)
	buffer.WriteString("<table cellspacing='0' cellpadding='2' border='1' bordercolor='black'>")
	buffer.WriteString("<tr><td align='right'><b>UID</b></td><td width='5'></td><td><b>Name</b></td></tr>")
	for i, v = range userDataList {
		buffer.WriteString(fmt.Sprintf("<tr><td align='right'>%d</td><td></td><td>%s</td></tr>",
			i, html.EscapeString(v.name)))
	}
	buffer.WriteString("</table>")
	buffer.WriteString(html_2)
	tpl_registeredUsersList = buffer.String()
}

//------------------------------------------------------------------------------
