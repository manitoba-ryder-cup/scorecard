package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tee-colors
func (r *Router) listTeeColors(w http.ResponseWriter, req *http.Request) {
	teeColors, err := r.CourseService.ListTeeColors(req.Context())
	if err != nil {
		respondError(req.Context(), w, http.StatusInternalServerError, "Failed to list tee colors", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeColors, toTeeColorDTO))
}

// POST /v1/tee-colors
func (r *Router) createTeeColor(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeAndValidate[sdk.CreateTeeColorRequest](w, req)
	if !ok {
		return
	}
	teeColor, err := r.CourseService.CreateTeeColor(req.Context(), golf.CreateTeeColorInput{Color: body.Color})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to create tee color", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeColorDTO(*teeColor))
}

// GET /v1/courses
func (r *Router) listCourses(w http.ResponseWriter, req *http.Request) {
	courses, err := r.CourseService.ListCourses(req.Context())
	if err != nil {
		respondError(req.Context(), w, http.StatusInternalServerError, "Failed to list courses", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(courses, toCourseDTO))
}

// GET /v1/courses/{id}
func (r *Router) getCourse(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "course")
	if !ok {
		return
	}
	course, err := r.CourseService.GetCourse(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to get course", err)
		return
	}
	respondJSON(w, http.StatusOK, toCourseDTO(*course))
}

// POST /v1/courses/{id}/tees
func (r *Router) addTeeSet(w http.ResponseWriter, req *http.Request) {
	courseID, ok := pathUUIDOr400(w, req, "id", "course")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.CreateTeeSetRequest](w, req)
	if !ok {
		return
	}
	holes := make([]golf.HoleInput, len(body.Holes))
	for i, h := range body.Holes {
		holes[i] = golf.HoleInput{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
	}
	teeSet, err := r.CourseService.CreateTeeSet(req.Context(), golf.CreateTeeSetInput{
		CourseID:   courseID,
		TeeColorID: body.TeeColorID,
		Slope:      body.Slope,
		Rating:     body.Rating,
		Holes:      holes,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to add tee set", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeSetDTO(*teeSet))
}

// GET /v1/courses/{id}/tees
// Lists a course's configured tee sets (with colour names) for match setup.
func (r *Router) listCourseTeeSets(w http.ResponseWriter, req *http.Request) {
	courseID, ok := pathUUIDOr400(w, req, "id", "course")
	if !ok {
		return
	}
	teeSets, err := r.CourseService.ListCourseTeeSets(req.Context(), courseID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list course tee sets", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeSets, toCourseTeeSetDTO))
}

// POST /v1/courses
func (r *Router) createCourse(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeAndValidate[sdk.CreateCourseRequest](w, req)
	if !ok {
		return
	}
	timeZone := body.TimeZone
	if timeZone == "" {
		timeZone = sdk.DefaultTimeZone
	}
	course, err := r.CourseService.CreateCourse(req.Context(), golf.CreateCourseInput{Name: body.Name, TimeZone: timeZone})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to create course", err)
		return
	}
	respondJSON(w, http.StatusCreated, toCourseDTO(*course))
}
