package apigrpc

import (
	"context"

	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"github.com/sangketkit01/simple-bank/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	violations := validCreateUserRequest(req)
	if violations != nil{
		return nil, invalidArguementError(violations)
	}
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password %s", err)
	}

	arg := db.CreateUserParams{
		Username:       req.GetUsername(),
		HashedPassword: hashedPassword,
		FullName:       req.GetFullName(),
		Email:          req.GetEmail(),
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			return nil, status.Errorf(codes.AlreadyExists, "username already exist: %s",err)
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s",err)
	}

	response := &pb.CreateUserResponse{
		User: convertUser(user),
	}
	return response, nil
}


func validCreateUserRequest(req *pb.CreateUserRequest) (violation []*errdetails.BadRequest_FieldViolation){
	if err := val.ValidateUsername(req.GetUsername()) ; err != nil{
		violation = append(violation, fieldViolation("username",err))
	}
	if err := val.ValidatePassword(req.GetPassword()) ; err != nil{
		violation = append(violation, fieldViolation("password",err))
	}
	if err := val.ValidateFullName(req.GetFullName()) ; err != nil{
		violation = append(violation, fieldViolation("full_name",err))
	}
	if err := val.ValidateEmail(req.GetEmail()) ; err != nil{
		violation = append(violation, fieldViolation("email",err))
	}

	return
}