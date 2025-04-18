package apigrpc

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"github.com/sangketkit01/simple-bank/val"
	"github.com/sangketkit01/simple-bank/worker"
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

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username: req.GetUsername(),
			HashedPassword: hashedPassword,
			FullName: req.GetFullName(),
			Email: req.GetEmail(),
		},
		AfterCreate: func(user db.User) error {
			taskPayload := &worker.PayloadSendVerifyEmail{
				Username: user.Username,
			}

			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue(worker.QueueCtitical),
			}
			return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
		},
	}

	txResult, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			return nil, status.Errorf(codes.AlreadyExists, "username already exist: %s",err)
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s",err)
	}

	// TODO: use db transaction

	response := &pb.CreateUserResponse{
		User: convertUser(txResult.User),
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