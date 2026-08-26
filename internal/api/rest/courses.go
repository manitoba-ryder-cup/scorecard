package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func (r *Router) listTeeColors(w http.ResponseWriter, req *http.Request) {
	teeColors, err := r.CourseService.ListTeeColors(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeColors, toTeeColorDTO))
}

func (r *Router) createTeeColor(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeAndValidate[sdk.CreateTeeColorRequest](w, req)
	if !ok {
		return
	}
	teeColor, err := r.CourseService.CreateTeeColor(req.Context(), golf.CreateTeeColorInput{Color: body.Color})
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeColorDTO(*teeColor))
}

func (r *Router) listCourses(w http.ResponseWriter, req *http.Request) {
	courses, err := r.CourseService.ListCourses(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(courses, toCourseDTO))
}

func (r *Router) getCourse(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "course")
	if !ok {
		return
	}
	course, err := r.CourseService.GetCourse(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, toCourseDTO(*course))
}

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
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeeSetDTO(*teeSet))
}

// listCourseTeeSets lists a course's configured tee sets (with colour names) for match setup.
func (r *Router) listCourseTeeSets(w http.ResponseWriter, req *http.Request) {
	courseID, ok := pathUUIDOr400(w, req, "id", "course")
	if !ok {
		return
	}
	teeSets, err := r.CourseService.ListCourseTeeSets(req.Context(), courseID)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teeSets, toCourseTeeSetDTO))
}

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
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toCourseDTO(*course))
}
