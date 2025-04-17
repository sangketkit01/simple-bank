package apigrpc

import (
	"context"
	"database/sql"
	"time"

	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"github.com/sangketkit01/simple-bank/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) UpdateUser(ctx context.Context, in *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error){
	authPayload, err := server.authorizaUser(ctx)
	if err != nil{
		return nil, unauthenticatedError(err)
	}

	violations := validUpdateUserRequest(in)
	if violations != nil{
		return nil, invalidArguementError(violations)
	}

	if authPayload.Username != in.Username{
		return nil, status.Errorf(codes.PermissionDenied, "cannot update other user info")
	}
	
	arg := db.UpdateUserParams{
		Username: in.Username,
		FullName: sql.NullString{
			String: in.GetFullName(),
			Valid: in.FullName != nil,
		},
		Email: sql.NullString{
			String: in.GetEmail(),
			Valid: in.Email != nil,
		},
	}

	if in.Password != nil{
		hashedPassword, err := util.HashPassword(in.GetPassword())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash password %s", err)
		}
		arg.HashedPassword = sql.NullString{
			String: hashedPassword,
			Valid: true,
		}

		arg.PasswordChangedAt = sql.NullTime{
			Time: time.Now(),
			Valid: true,
		}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %s",err)
	}

	response := &pb.UpdateUserResponse{
		User: convertUser(user),
	}
	return response, nil
}

func validUpdateUserRequest(req *pb.UpdateUserRequest) (violation []*errdetails.BadRequest_FieldViolation){
	if err := val.ValidateUsername(req.GetUsername()) ; err != nil{
		violation = append(violation, fieldViolation("username",err))
	}

	if req.Password != nil{
		if err := val.ValidatePassword(req.GetPassword()) ; err != nil{
			violation = append(violation, fieldViolation("password",err))
		}
	}

	if req.FullName != nil{
		if err := val.ValidateFullName(req.GetFullName()) ; err != nil{
			violation = append(violation, fieldViolation("full_name",err))
		}
	}

	if req.Email != nil{
		if err := val.ValidateEmail(req.GetEmail()) ; err != nil{
			violation = append(violation, fieldViolation("email",err))
		}
	}

	return
}