package frontend

import (
	"fmt"
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

// NOTE: Jobs are restricted by Machine UUID

// JobResponse return on a successful GET, PUT, PATCH or POST of a single Job
// swagger:response
type JobResponse struct {
	// in: body
	Body *models.Job
}

// JobsResponse return on a successful GET of all Jobs
// swagger:response
type JobsResponse struct {
	// in: body
	Body []*models.Job
}

// JobActionsResponse return on a successful GET of a Job's actions
// swagger:response
type JobActionsResponse struct {
	// in: body
	Body []*models.JobAction
}

// JobParamsResponse return on a successful GET of all Job's Params
// swagger:response
type JobParamsResponse struct {
	// in: body
	Body map[string]interface{}
}

// This is a HACK - I can't figure out how to get
// swagger to render this a binary.  So we lie.
// We also override this object from the server
// directory to have a binary format which
// turns it into a stream.
//
// JobLogResponse returned on a successful GET of a log
// swagger:response
type JobLogResponse struct {
	// in: body
	// format: binary
	Body string
}

// JobBodyParameter used to inject a Job
// swagger:parameters createJob putJob
type JobBodyParameter struct {
	// in: body
	// required: true
	Body *models.Job
}

// JobPatchBodyParameter used to patch a Job
// swagger:parameters patchJob
type JobPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// JobLogBodyParameter used to append to a Job log
// swagger:parameters putJobLog
type JobLogPutBodyParameter struct {
	// in: body
	// required: true
	Body interface{}
}

// JobPathParameter used to find a Job in the path
// swagger:parameters putJobs getJob putJob patchJob deleteJob getJobParams postJobParams getJobActions getJobLog putJobLog headJob
type JobPathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
}

// JobParamsBodyParameter used to set Job Params
// swagger:parameters postJobParams
type JobParamsBodyParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// JobListPathParameter used to limit lists of Job by path options
// swagger:parameters listJobs listStatsJobs
type JobListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Available string
	// in: query
	Valid string
	// in: query
	ReadOnly string
	// in: query
	Uuid string
	// in: query
	Stage string
	// in: query
	Task string
	// in: query
	State string
	// in: query
	Machine string
	// in: query
	Archived string
	// in: query
	StartTime string
	// in: query
	EndTime string
}

func (f *Frontend) InitJobApi() {
	// swagger:route GET /jobs Jobs listJobs
	//
	// Lists Jobs filtered by some parameters.
	//
	// This will show all Jobs by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Uuid = string
	//    Stage = string
	//    Task = string
	//    State = string
	//    Machine = string
	//    Archived = boolean
	//    StartTime = datetime
	//    EndTime = datetime
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
	//
	// Functions:
	//    Eq(value) = Return items that are equal to value
	//    Lt(value) = Return items that are less than value
	//    Lte(value) = Return items that less than or equal to value
	//    Gt(value) = Return items that are greater than value
	//    Gte(value) = Return items that greater than or equal to value
	//    Between(lower,upper) = Return items that are inclusively between lower and upper
	//    Except(lower,upper) = Return items that are not inclusively between lower and upper
	//
	// Example:
	//    Uuid=fred - returns items named fred
	//    Uuid=Lt(fred) - returns items that alphabetically less than fred.
	//    Uuid=Lt(fred)&Archived=true - returns items with Uuid less than fred and Archived is true
	//
	// Responses:
	//    200: JobsResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/jobs",
		func(c *gin.Context) {
			f.List(c, &backend.Job{})
		})

	// swagger:route HEAD /jobs Jobs listStatsJobs
	//
	// Stats of the List Jobs filtered by some parameters.
	//
	// This will return headers with the stats of the list.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Uuid = string
	//    Stage = string
	//    Task = string
	//    State = string
	//    Machine = string
	//    Archived = boolean
	//    StartTime = datetime
	//    EndTime = datetime
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
	//
	// Functions:
	//    Eq(value) = Return items that are equal to value
	//    Lt(value) = Return items that are less than value
	//    Lte(value) = Return items that less than or equal to value
	//    Gt(value) = Return items that are greater than value
	//    Gte(value) = Return items that greater than or equal to value
	//    Between(lower,upper) = Return items that are inclusively between lower and upper
	//    Except(lower,upper) = Return items that are not inclusively between lower and upper
	//
	// Example:
	//    Uuid=fred - returns items named fred
	//    Uuid=Lt(fred) - returns items that alphabetically less than fred.
	//    Uuid=Lt(fred)&Archived=true - returns items with Uuid less than fred and Archived is true
	//
	// Responses:
	//    200: NoContentResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.HEAD("/jobs",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Job{})
		})

	// swagger:route POST /jobs Jobs createJob
	//
	// Create a Job
	//
	// Create a Job from the provided object, Only Machine and UUID are used.
	//
	//     Responses:
	//       201: JobResponse
	//       202: JobResponse
	//       204: NoContentResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	//       500: ErrorResponse
	f.ApiGroup.POST("/jobs",
		func(c *gin.Context) {
			// We don't use f.Create() because we need to be able to assign random
			// UUIDs to new Jobs without forcing the client to do so, yet allow them
			// for testing purposes amd if they alrady have a UUID scheme for jobs.
			b := &backend.Job{}
			if !assureDecode(c, b) {
				return
			}
			if b.Machine == nil {
				c.JSON(http.StatusBadRequest, models.NewError(c.Request.Method, http.StatusBadRequest, "Create request must have Machine field"))
				return
			}
			if !f.assureAuth(c, "jobs", "create", b.AuthKey()) {
				return
			}
			if b.Uuid == nil || len(b.Uuid) == 0 {
				b.Uuid = uuid.NewRandom()
			}
			var res models.Model
			var err error
			rt := f.rt(c, b.Locks("create")...)
			var code int
			rt.Do(func(d backend.Stores) {
				mo := rt.Find("machines", b.Machine.String())
				if mo == nil {
					err = &models.Error{Code: http.StatusUnprocessableEntity, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Machine %s does not exist", b.Machine.String())}}
					code = http.StatusUnprocessableEntity
					return
				}
				m := backend.AsMachine(mo)

				// Machine isn't runnable return conflict
				if !m.Runnable {
					err = &models.Error{Code: http.StatusConflict, Type: "Conflict",
						Messages: []string{fmt.Sprintf("Machine %s is not runnable", b.Machine.String())}}
					code = http.StatusConflict
					return
				}

				// Are we running a job or not on list yet, do some checking.
				newCT := m.CurrentTask
				if newCT < len(m.Tasks) {
					if jo := rt.Find("jobs", m.CurrentJob.String()); jo != nil && newCT != -1 {
						cj := jo.(*backend.Job)
						if cj.State == "failed" {
							// We are re-running the current task
						} else if cj.State == "finished" {
							// We are running the next task
							newCT += 1
						} else if cj.State == "incomplete" {
							b = cj
							code = http.StatusAccepted
							return
						} else {
							// Need to error - running job already running or just created.
							err = &models.Error{Code: http.StatusConflict, Type: "Conflict",
								Messages: []string{fmt.Sprintf("Machine %s already has running or created job", b.Machine.String())}}
							code = http.StatusConflict
							return
						}
					} else if jo != nil {
						// At this point, newCT == -1 (we are starting over on a list)
						// We have an old job - check its state.
						// We could have been forced and need to close out a job.
						// if it is running, created, or incomplete, we need
						// to mark it failed, but leave us runnable.
						cj := jo.(*backend.Job)
						if cj.State == "running" || cj.State == "created" || cj.State == "incomplete" {
							cj.State = "failed"
							if _, err = rt.Update(cj); err != nil {
								code = http.StatusBadRequest
								return
							}
							m.Runnable = true
						}
						newCT += 1

					} else {
						// No current job. Index to next.
						newCT += 1
					}
				}

				if newCT >= len(m.Tasks) {
					// Nothing to do.
					if newCT != m.CurrentTask {
						m.CurrentTask = newCT
						_, err = rt.Save(m)
						if err != nil {
							code = http.StatusInternalServerError
							return
						}

					}
					code = http.StatusNoContent
					return
				}

				// Fill in new job.
				b.State = "created"
				if m.CurrentJob == nil {
					b.Previous = uuid.Parse("00000000-0000-0000-0000-000000000000")
				} else {
					b.Previous = m.CurrentJob
				}
				b.Stage = m.Stage
				b.Task = m.Tasks[newCT]
				// Create the job, and then update the machine
				_, err = rt.Create(b)
				if err == nil {
					m.CurrentTask = newCT
					m.CurrentJob = b.Uuid
					_, err = rt.Save(m)
					if err != nil {
						rt.Remove(b)
						code = http.StatusBadRequest
						return
					}
				}
				code = http.StatusCreated
			})
			if err != nil {
				be, ok := err.(*models.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, models.NewError(c.Request.Method, http.StatusBadRequest, err.Error()))
				}
			} else if code == http.StatusNoContent {
				c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
			} else {
				s, ok := models.Model(b).(Sanitizable)
				if ok {
					res = s.Sanitize()
				} else {
					res = b
				}
				c.JSON(code, res)
			}
		})

	// swagger:route GET /jobs/{uuid} Jobs getJob
	//
	// Get a Job
	//
	// Get the Job specified by {uuid} or return NotFound.
	//
	//     Responses:
	//       200: JobResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/jobs/:uuid",
		func(c *gin.Context) {
			f.Fetch(c, &backend.Job{}, c.Param(`uuid`))
		})

	// swagger:route HEAD /jobs/{uuid} Jobs headJob
	//
	// See if a Job exists
	//
	// Return 200 if the Job specifiec by {uuid} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/jobs/:uuid",
		func(c *gin.Context) {
			f.Exists(c, &backend.Job{}, c.Param(`uuid`))
		})

	// swagger:route PATCH /jobs/{uuid} Jobs patchJob
	//
	// Patch a Job
	//
	// Update a Job specified by {uuid} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: JobResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/jobs/:uuid",
		func(c *gin.Context) {
			f.Patch(c, &backend.Job{}, c.Param(`uuid`))
		})

	// swagger:route PUT /jobs/{uuid} Jobs putJob
	//
	// Put a Job
	//
	// Update a Job specified by {uuid} using a JSON Job
	//
	//     Responses:
	//       200: JobResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/jobs/:uuid",
		func(c *gin.Context) {
			f.Update(c, &backend.Job{}, c.Param(`uuid`))
		})

	// swagger:route DELETE /jobs/{uuid} Jobs deleteJob
	//
	// Delete a Job
	//
	// Delete a Job specified by {uuid}
	//
	//     Responses:
	//       200: JobResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/jobs/:uuid",
		func(c *gin.Context) {
			f.Remove(c, &backend.Job{}, c.Param(`uuid`))
		})

	// swagger:route GET /jobs/{uuid}/actions Jobs getJobActions
	//
	// Get actions for this job
	//
	// Get actions for the Job specified by {uuid} or return NotFound.
	//
	//     Responses:
	//       200: JobActionsResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.GET("/jobs/:uuid/actions",
		func(c *gin.Context) {
			uuid := c.Param(`uuid`)
			j := &backend.Job{}
			var bad bool
			var err error
			rt := f.rt(c, j.Locks("actions")...)
			rt.Do(func(d backend.Stores) {
				var jo models.Model
				if jo = rt.Find("jobs", uuid); jo == nil {
					err = &models.Error{Code: http.StatusNotFound, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Job %s does not exist", uuid)}}
					bad = true
					return
				}
				j = backend.AsJob(jo)
			})
			if bad {
				c.JSON(err.(*models.Error).Code, err)
				return
			}

			if !f.assureAuth(c, "jobs", "actions", j.AuthKey()) {
				return
			}
			actions, err := j.RenderActions(rt)
			if err != nil {
				be, ok := err.(*models.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, models.NewError(c.Request.Method, http.StatusBadRequest, err.Error()))
				}
			}
			c.JSON(http.StatusOK, actions)

		})

	// swagger:route GET /jobs/{uuid}/log Jobs getJobLog
	//
	// Get the log for this job
	//
	// Get log for the Job specified by {uuid} or return NotFound.
	//
	//     Produces:
	//       application/octet-stream
	//       application/json
	//
	//     Responses:
	//       200: JobLogResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/jobs/:uuid/log",
		func(c *gin.Context) {
			uuid := c.Param(`uuid`)
			j := &backend.Job{}
			var bad bool
			var err *models.Error
			rt := f.rt(c, j.Locks("get")...)
			rt.Do(func(d backend.Stores) {
				var jo models.Model
				if jo = rt.Find("jobs", uuid); jo == nil {
					err = &models.Error{Code: http.StatusNotFound, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Job %s does not exist", uuid)}}
					bad = true
					return
				}
				j = backend.AsJob(jo)
			})
			if bad {
				c.JSON(err.Code, err)
				return
			}

			if !f.assureAuth(c, "jobs", "log", j.AuthKey()) {
				return
			}

			c.Writer.Header().Set("Content-Type", "application/octet-stream")
			c.File(j.LogPath())
		})

	// swagger:route PUT /jobs/{uuid}/log Jobs putJobLog
	//
	// Append the string to the end of the job's log.
	//     Consumes:
	//       application/octet-stream
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       204: NoContentResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       415: ErrorResponse
	//       404: ErrorResponse
	//       500: ErrorResponse
	f.ApiGroup.PUT("/jobs/:uuid/log",
		func(c *gin.Context) {
			if c.Request.Body == nil {
				err := &models.Error{Code: http.StatusBadRequest}
				c.JSON(err.Code, err)
				return
			}
			defer c.Request.Body.Close()
			if c.Request.Header.Get(`Content-Type`) != `application/octet-stream` {
				c.JSON(http.StatusUnsupportedMediaType,
					models.NewError("API ERROR", http.StatusUnsupportedMediaType,
						"job log put must have content-type application/octet-stream"))
				return
			}
			uuid := c.Param(`uuid`)
			j := &backend.Job{}
			var bad bool
			var err *models.Error
			rt := f.rt(c, j.Locks("get")...)
			rt.Do(func(d backend.Stores) {
				var jo models.Model
				if jo = d("jobs").Find(uuid); jo == nil {
					err = &models.Error{Code: http.StatusNotFound, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Job %s does not exist", uuid)}}
					bad = true
					return
				}
				j = backend.AsJob(jo)
			})
			if bad {
				c.JSON(err.Code, err)
				return
			}

			if !f.assureAuth(c, "jobs", "log", j.AuthKey()) {
				return
			}

			if err := j.Log(rt, c.Request.Body); err != nil {
				err2 := &models.Error{Code: http.StatusInternalServerError, Type: "Server ERROR",
					Messages: []string{err.Error()}}
				c.JSON(err2.Code, err2)
			} else {
				c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
			}
		})

}
