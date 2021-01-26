package profile_v1

import (
	"fmt"
	"google.golang.org/grpc/codes"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type ErrProfileNotFound struct {
	Id uint64
}

func (error ErrProfileNotFound) GRPCStatus() *status.Status {
	errStatus := status.New(codes.NotFound, fmt.Sprintf("profile not found: %d", error.Id))
	errMsg := fmt.Sprintf("The requested profile does not exist: %d", error.Id)
	errDetails := &errdetails.LocalizedMessage{Locale: "en-US", Message: errMsg}
	std, err := errStatus.WithDetails(errDetails)
	if err != nil {
		return errStatus
	}
	return std
}

func (error ErrProfileNotFound) Error() string {
	return error.GRPCStatus().Err().Error()
}
