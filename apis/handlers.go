package apis

import "es_middleware/models"

type ESMiddlewareInterface interface {
	ESSearch(request ESSearchRequest, response *ESSearchResponse) error
	ESBulkInsert(request ESBulkInsertRequest, response *ESBulkInsertResponse) error
	ESBulkDelete(request ESBulkDeleteRequest, response *ESBulkDeleteResponse) error
	ESDelete(request ESDeleteRequest, response *ESDeleteResponse) error
}

type Handler struct{}

type ESSearchRequest struct {
	keyword string
	size    int
	offset  int
}

type ESSearchResponse struct {
	floors models.Floors
	err    error
}

type ESBulkInsertRequest struct {
	floors []models.FloorModel
}

type ESBulkInsertResponse struct {
	err error
}

type ESBulkDeleteRequest struct {
	floorIDs []int
}

type ESBulkDeleteResponse struct {
	err error
}

type ESDeleteRequest struct {
	floorID int
}

type ESDeleteResponse struct {
	err error
}

func (handler Handler) ESSearch(request ESSearchRequest, response *ESSearchResponse) error {
	response.floors, response.err = models.Search(request.keyword, request.size, request.offset)
	return response.err
}

func (handler Handler) ESBulkInsert(request ESBulkInsertRequest, response *ESBulkInsertResponse) error {
	response.err = models.BulkInsert(request.floors)
	return response.err
}

func (handler Handler) ESBulkDelete(request ESBulkDeleteRequest, response *ESBulkDeleteResponse) error {
	response.err = models.BulkDelete(request.floorIDs)
	return response.err
}

func (handler Handler) ESDelete(request ESDeleteRequest, response *ESDeleteResponse) error {
	response.err = models.FloorDelete(request.floorID)
	return response.err
}
