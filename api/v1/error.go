package profile_v1

import (
	"fmt"
	"google.golang.org/grpc/codes"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type ErrProfileNotFound struct {
	Id string
}

func (error ErrProfileNotFound) GRPCStatus() *status.Status {
	errStatus := status.New(codes.NotFound, fmt.Sprintf("%s not found", error.Id))
	errMsg := fmt.Sprintf("The requested profile does not exist: %s", error.Id)
	errDetails := &errdetails.LocalizedMessage{Locale: "en-US", Message: errMsg}
	errStatusDetails, err := errStatus.WithDetails(errDetails)
	if err != nil {
		return errStatus
	}
	return errStatusDetails
}

func (error ErrProfileNotFound) Error() string {
	return error.GRPCStatus().Err().Error()
}
