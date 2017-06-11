// asq.go

package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/llgcode/draw2d/draw2dimg"
)

//------------------------------------------------------------------------------

// Lists
type tAntiSpamQuestion struct {
	timeOfCreation int64  // Time of Creation of the Question
	question       []byte // Question's Contents, binary Image
	answer         uint8  // Correct Answer to the Question
}
type tAntiSpamQuestions map[uint64]tAntiSpamQuestion // Key = QID

type tAsqJob struct {
	qid           uint64
	asq           tAntiSpamQuestion
	returnChannel chan tAsqJob
	result        bool
	action        uint8
}

//------------------------------------------------------------------------------

const asqRevisorIntervalDefault = 120 // Interval for Monitoring Anti-Spam Questions, in Seconds
const asqTimeout = 60                 // Questions older than this Value are thrown out, in Seconds
const asqManagerChanBufferLen = 64    // Buffer Length of the ASQ Manager's Channel
const asqJobGet = 1                   // Action Code for ASQ Manager to Get ASQ
const asqJobSet = 2                   // Action Code for ASQ Manager to Set ASQ
const asqJobDelete = 3                // Action Code for ASQ Manager to Delete ASQ
const asqJobClearData = 4             // Action Code for ASQ Manager to Clear Question in ASQ

//------------------------------------------------------------------------------

// Internal Parameters
var asqRevisorInterval int

// Lists
var asqsList tAntiSpamQuestions

// Channels
var asqRevisorQuit chan int
var asqManagerChan chan tAsqJob
var asqManagerQuit chan int

//------------------------------------------------------------------------------

func asq_create() (qid uint64) {

	// Creates Anti-Spam Question.

	var asq *tAntiSpamQuestion
	var job *tAsqJob
	var exists bool
	var rcvChan chan tAsqJob

	// Create a Question
	asq = new(tAntiSpamQuestion)
	asq_createData(asq)

	for {

		// Random Key for Map, Must be Unique
		exists = true
		for {

			qid = generateRandomUint64()
			_, exists = asqsList[qid]
			if !exists {
				break
			}
		}

		// Set ASQ
		// Create Job
		rcvChan = make(chan tAsqJob)
		job = new(tAsqJob)
		job.action = asqJobSet // Set
		job.qid = qid
		job.asq = *asq
		job.returnChannel = rcvChan

		// Send Job
		asqManagerChan <- *job

		// Wait for Feedback
		*job = <-rcvChan

		if job.result == true {
			break
		}
	}

	return job.qid
}

//------------------------------------------------------------------------------

func asq_createData(asq *tAntiSpamQuestion) {

	// Creates the Question (Image, binary Data) and returns it with the answer.

	var img_width int = 200
	var img_height int = 200
	var minDim int
	var img *image.RGBA
	var gc *draw2dimg.GraphicContext

	if img_width > img_height {
		minDim = img_height
	} else {
		minDim = img_width
	}

	img = image.NewRGBA(image.Rect(0, 0, img_width, img_height))
	gc = draw2dimg.NewGraphicContext(img)

	var i, n, op, op_min, op_rnd uint8
	var xc, yc, xc_min, xc_rnd, yc_min, yc_rnd int
	var r, r_min, r_rnd int
	var lt, lt_min, lt_rnd, op_rnd_int int
	var col color.Color
	var buf *bytes.Buffer
	var encoder *png.Encoder

	n = uint8(3 + rand.Intn(3)) // [3;5]

	xc_min = img_width / 4
	xc_rnd = img_width / 2

	yc_min = img_height / 4
	yc_rnd = img_height / 2

	r_min = 15
	r_rnd = (minDim / 3) - r_min

	lt_min = 5
	lt_rnd = 4

	op_min = 95
	op_rnd_int = math.MaxUint8 + 1 - int(op_min)
	op_rnd = uint8(op_rnd_int)

	for i = 1; i <= n; i++ {
		op = op_min + uint8(rand.Intn(int(op_rnd)))
		col = color.RGBA{generateRandomUint8(), generateRandomUint8(), generateRandomUint8(), op}
		gc.SetFillColor(col)
		gc.SetStrokeColor(col)
		r = r_min + rand.Intn(r_rnd)
		xc = xc_min + rand.Intn(xc_rnd)
		yc = yc_min + rand.Intn(yc_rnd)
		lt = lt_min + rand.Intn(lt_rnd)
		gc.SetFillColor(col)
		gc.SetStrokeColor(col)
		gc.SetLineWidth(float64(lt))
		gc.BeginPath()
		gc.ArcTo(float64(xc), float64(yc), float64(r), float64(r), 0, 2*math.Pi)
		gc.Close()
		gc.Stroke()
	}

	// Save to Buffer
	buf = new(bytes.Buffer)
	encoder = new(png.Encoder)
	encoder.CompressionLevel = png.BestCompression
	encoder.Encode(buf, img)

	// Fill Data in asq
	(*asq).answer = n
	(*asq).question = buf.Bytes()
}

//------------------------------------------------------------------------------

func asq_clearQuestionData(qid uint64) {

	// Clears Data (Image) of an Anti-Spam Question.

	var rcvChan chan tAsqJob
	var asqJob *tAsqJob

	// Create Job
	rcvChan = make(chan tAsqJob)
	asqJob = new(tAsqJob)
	asqJob.action = asqJobClearData // CQD
	asqJob.qid = qid
	asqJob.returnChannel = rcvChan

	// Send Job
	asqManagerChan <- *asqJob

	// Wait for Feedback
	*asqJob = <-rcvChan
}

//------------------------------------------------------------------------------

func asqRevisor() {

	// ASQ Revisor periodically checks created Anti-Spam Questions
	// and deletes outdated ones.

	var toc, now, criterion int64 // Time of Creation of the Question
	var loop bool = true
	var i uint64
	var v tAntiSpamQuestion
	var rcvChan chan tAsqJob
	var asqJob *tAsqJob

	// Preparations
	rcvChan = make(chan tAsqJob)
	asqJob = new(tAsqJob)
	asqJob.action = asqJobDelete // Delete
	asqJob.returnChannel = rcvChan

	// Periodical Check for active Clients
	for loop {

		// Starting Job
		now = time.Now().Unix()
		criterion = now - asqTimeout

		// For Each active Client
		for i, v = range asqsList {

			toc = v.timeOfCreation
			if toc < criterion {
				// Outdated Question must be deleted from the List

				// Re-Configure Job for specific Client
				asqJob.qid = i

				// Send Job
				asqManagerChan <- *asqJob

				// Wait for Feedback
				*asqJob = <-rcvChan

			}
		}

		// Checking for Stop Signal
		select {
		case <-asqRevisorQuit:
			loop = false
			log.Println("Closing ASQ Revisor...") //
		default:

		}

		// Wait for next Job
		time.Sleep(time.Second * time.Duration(asqRevisorInterval))
	}
}

//------------------------------------------------------------------------------

func asqManager() {

	// Manages Anti Spam Question Requests.

	var loop bool = true
	var job *tAsqJob
	var exists bool
	var tmp_asq *tAntiSpamQuestion

	job = new(tAsqJob)

	for loop {

		// Get Job from Channel
		*job = <-asqManagerChan

		if job.action == asqJobSet {

			// Add an ASQ to the List of ASQs (if it does not already exist).
			// QID in Job shows which ASQ to create.
			// Fields "answer" and "quesion" are provided by the Sender.

			// Set
			_, exists = asqsList[job.qid]
			if !exists {
				job.asq.timeOfCreation = time.Now().Unix()
				asqsList[job.qid] = job.asq
				job.result = true
			} else {
				job.result = false
			}

		} else if job.action == asqJobGet {

			// Get an ASQ from the ASQ List (if it exists).
			// QID in Job shows which ASQ to return to Sender.
			// Manager returns asq in Job.

			// Get
			_, exists = asqsList[job.qid]
			if exists {
				job.asq = asqsList[job.qid]

				job.result = true

			} else {
				job.result = false
			}

		} else if job.action == asqJobDelete {

			// Delete an ASQ from the ASQ List.
			// QID in Job shows which ASQ to delete.

			// Delete
			_, exists = asqsList[job.qid]
			if exists {
				delete(asqsList, job.qid)
			}
			job.result = true

		} else if job.action == asqJobClearData {

			// Clear Question Data from the ASQ (to save Space).
			// The Answer is not cleared.
			// QID in Job shows which ASQ to clear. Other Fields are not known,
			// are taken from asqsList and returned to Sender.

			// Clear Question Data
			_, exists = asqsList[job.qid]
			if exists {

				tmp_asq = new(tAntiSpamQuestion)
				tmp_asq.answer = asqsList[job.qid].answer                 // save Answer
				tmp_asq.question = []byte{}                               // "clearing" Data
				tmp_asq.timeOfCreation = asqsList[job.qid].timeOfCreation // save toc
				asqsList[job.qid] = *tmp_asq
				job.result = true
			} else {
				job.result = false
			}
		}

		job.returnChannel <- *job // Send back

		// Checking for Stop Signal
		select {
		case <-asqManagerQuit:
			loop = false
			log.Println("Closing ASQ Manager...") //
		default:
		}
	}
}

//------------------------------------------------------------------------------
