package frontend

import (
	"fmt"
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

// JobResponse return on a successful GET, PUT, PATCH or POST of a single Job
// swagger:response
type JobResponse struct {
	// in: body
	Body *backend.Job
}

// JobsResponse return on a successful GET of all Jobs
// swagger:response
type JobsResponse struct {
	// in: body
	Body []*backend.Job
}

// JobActionsResponse return on a successful GET of a Job's actions
// swagger:response
type JobActionsResponse struct {
	// in: body
	Body []*backend.JobAction
}

// JobParamsResponse return on a successful GET of all Job's Params
// swagger:response
type JobParamsResponse struct {
	// in: body
	Body map[string]interface{}
}

// JobBodyParameter used to inject a Job
// swagger:parameters createJob putJob
type JobBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Job
}

// JobPatchBodyParameter used to patch a Job
// swagger:parameters patchJob
type JobPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// JobPathParameter used to find a Job in the path
// swagger:parameters putJobs getJob putJob patchJob deleteJob getJobParams postJobParams getJobActions
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
// swagger:parameters listJobs
type JobListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Uuid string
	// in: query
	BootEnv string
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
	//    BootEnv = string
	//    Task = string
	//    State = string
	//    Machine = string
	//    Archived = boolean
	//    StartTime = datetime
	//    EndTime = datetime
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
			f.List(c, f.dt.NewJob())
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
			if !assureAuth(c, f.Logger, "jobs", "create", "*") {
				return
			}
			b := f.dt.NewJob()
			if !assureDecode(c, b) {
				return
			}
			if b.Uuid == nil || len(b.Uuid) == 0 {
				b.Uuid = uuid.NewRandom()
			}
			var res store.KeySaver
			var err error
			code := func() int {
				d, unlocker := f.dt.LockEnts(store.KeySaver(b).(Lockable).Locks("create")...)
				defer unlocker()

				// See the backend.Job comments for what is going on.
				machines := d("machines")
				jobs := d("jobs")

				var mo store.KeySaver
				// XXX?? Should find return a live object.
				if mo = machines.Find(b.Machine.String()); mo == nil {
					err = &backend.Error{Code: http.StatusUnprocessableEntity, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Machine %s does not exist", b.Machine.String())}}
					return http.StatusUnprocessableEntity
				}
				m := backend.AsMachine(mo)

				// Machine isn't runnable return conflict
				if !m.Runnable {
					err = &backend.Error{Code: http.StatusConflict, Type: "Conflict",
						Messages: []string{fmt.Sprintf("Machine %s is not runnable", b.Machine.String())}}
					return http.StatusConflict
				}

				// Are we running a job or not on list yet, do some checking.
				newCT := m.CurrentTask
				if newCT < len(m.Tasks) {
					if jo := jobs.Find(m.CurrentJob.String()); jo != nil && newCT != -1 {
						cj := backend.AsJob(jo)
						if cj.State == "failed" {
							// We are re-running the current task
						} else if cj.State == "finished" {
							// We are running the next task
							newCT += 1
						} else if cj.State == "incomplete" {
							b = cj
							return http.StatusAccepted
						} else {
							// Need to error - running job already running or just created.
							err = &backend.Error{Code: http.StatusConflict, Type: "Conflict",
								Messages: []string{fmt.Sprintf("Machine %s already has running or created job", b.Machine.String())}}
							return http.StatusConflict
						}
					} else if jo != nil {
						// We have an old job, but we are starting over.  Mark it failed
						cj := backend.AsJob(jo)
						cj.State = "failed"
						if _, err = f.dt.Update(d, cj, nil); err != nil {
							return http.StatusBadRequest
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
						_, err = f.dt.Update(d, m, nil)
						if err != nil {
							return http.StatusInternalServerError
						}

					}
					return http.StatusNoContent
				}

				// Fill in new job.
				b.State = "created"
				if m.CurrentJob == nil {
					b.Previous = uuid.Parse("00000000-0000-0000-0000-000000000000")
				} else {
					b.Previous = m.CurrentJob
				}
				b.BootEnv = m.BootEnv
				b.Task = m.Tasks[newCT]

				// GREG: Setup log path

				// Create the job, and then create the machine
				_, err = f.dt.Create(d, b, nil)
				if err == nil {
					m.CurrentTask = newCT
					m.CurrentJob = b.Uuid
					_, err = f.dt.Update(d, m, nil)
					if err != nil {
						_, err = f.dt.Remove(d, b, nil)
					}
				}

				return http.StatusCreated

			}()
			if err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else if code == http.StatusNoContent {
				c.JSON(code, nil)
			} else {
				s, ok := store.KeySaver(b).(Sanitizable)
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
			f.Fetch(c, f.dt.NewJob(), c.Param(`uuid`))
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
			f.Patch(c, f.dt.NewJob(), c.Param(`uuid`), nil)
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
			f.Update(c, f.dt.NewJob(), c.Param(`uuid`), nil)
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
			b := f.dt.NewJob()
			b.Uuid = uuid.Parse(c.Param(`uuid`))
			f.Remove(c, b, nil)
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
			// We don't use f.Create() because we need to be able to assign random
			// UUIDs to new Jobs without forcing the client to do so, yet allow them
			// for testing purposes amd if they alrady have a UUID scheme for jobs.
			if !assureAuth(c, f.Logger, "jobs", "actions", c.Param(`uuid`)) {
				return
			}
			uuid := c.Param(`uuid`)
			j := f.dt.NewJob()
			bad := func() bool {
				d, unlocker := f.dt.LockEnts(store.KeySaver(j).(Lockable).Locks("actions")...)
				defer unlocker()

				var jo store.KeySaver
				if jo = d("jobs").Find(uuid); jo == nil {
					err := &backend.Error{Code: http.StatusNotFound, Type: backend.ValidationError,
						Messages: []string{fmt.Sprintf("Job %s does not exist", uuid)}}
					c.JSON(err.Code, err)
					return true
				}
				j = backend.AsJob(jo)
				return false
			}()
			if bad {
				return
			}

			var err error
			actions, err := j.RenderActions()
			if err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			}
			c.JSON(http.StatusOK, actions)

		})
}
