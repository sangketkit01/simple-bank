package apigrpc

import (
	"context"
	"database/sql"

	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"github.com/sangketkit01/simple-bank/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	violations := validLoginUserRequest(req)
	if violations != nil{
		return nil, invalidArguementError(violations)
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "User not found: %s", err)
		}

		return nil, status.Errorf(codes.Internal, "cannot get user: %s", err)
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "incorrect password: %s", err)
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create token: %s", err)
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create refresh token: %s", err)
	}

	mtdt := server.extractMetadata(ctx)

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    mtdt.UserAgent,
		ClientIp:     mtdt.ClientIP,
		IsBlock:      false,
		ExpiredAt:    refreshPayload.ExpiredAt,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create session: %s", err)
	}

	response := &pb.LoginUserResponse{
		User:                convertUser(user),
		SessionId:           session.ID.String(),
		AccessToken:         accessToken,
		AccessTokenExpired:  timestamppb.New(accessPayload.ExpiredAt),
		RefreshToken:        refreshToken,
		RefreshTokenExpired: timestamppb.New(refreshPayload.ExpiredAt),
	}

	return response, nil
}

func validLoginUserRequest(req *pb.LoginUserRequest) (violation []*errdetails.BadRequest_FieldViolation){
	if err := val.ValidateUsername(req.GetUsername()) ; err != nil{
		violation = append(violation, fieldViolation("username",err))
	}
	if err := val.ValidatePassword(req.GetPassword()) ; err != nil{
		violation = append(violation, fieldViolation("password",err))
	}

	return
}
