package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.24

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v4"
	"strconv"
	"strings"
	"test-server-go/graph/model"
	"test-server-go/internal/argon2"
	"test-server-go/internal/mailer"
	"test-server-go/internal/models"
	"test-server-go/internal/token"
	"test-server-go/internal/tools"
	v "test-server-go/internal/validator"
)

// AuthSignupWithoutCode is the resolver for the authSignupWithoutCode field.
func (r *mutationResolver) AuthSignupWithoutCode(ctx context.Context, input model.SignupWithoutCodeInput) (bool, error) {
	// Block 1 - data validation
	nickname := strings.TrimSpace(input.Nickname)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)

	if err := v.Validate(nickname, v.IsMinMaxLen(5, 32), v.IsContainsSpace()); err != nil {
		return false, errors.New("nickname: " + err.Error())
	}
	if err := v.Validate(email, v.IsMinMaxLen(6, 64), v.IsContainsSpace(), v.IsEmail()); err != nil {
		return false, errors.New("email: " + err.Error())
	}
	if err := v.Validate(password, v.IsMinMaxLen(6, 64), v.IsContainsSpace()); err != nil {
		return false, errors.New("password: " + err.Error())
	}
	emailDomainExists, err := mailer.CheckEmailDomainExistence(email)
	if !emailDomainExists {
		return false, errors.New("email: the email domain is not exist")
	}
	if err != nil {
		r.App.Logrus.NewWarn("error in checked the email domain: " + err.Error())
	}

	// Block 2 - checking for an existing nickname and email
	result := execInTx(ctx, r.App.Postgres.Pool, r.App.Logrus, func(tx pgx.Tx) (interface{}, error) {
		var result models.ExistsNicknameEmail
		localErr := tx.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM account.user WHERE nickname = $1)::boolean AS username_exists, EXISTS(SELECT 1 FROM account.user WHERE email = $2)::boolean AS email_exists",
			nickname, email).Scan(&result.NicknameExists, &result.EmailExists)
		return result, localErr
	})
	existsAccount := result.(models.ExistsNicknameEmail)
	if existsAccount.NicknameExists {
		return false, errors.New("this nickname is already in use")
	}
	if existsAccount.EmailExists {
		return false, errors.New("this email is already in use")
	}

	// Block 3 - generating code and inserting a temporary account record
	confirmCode, err := tools.GenerateConfirmationCode()
	if err != nil {
		r.App.Logrus.NewError("the confirm code not generated", err)
	}

	result = execInTx(ctx, r.App.Postgres.Pool, r.App.Logrus, func(tx pgx.Tx) (interface{}, error) {
		var resultRegistrationTempNo bool
		err = tx.QueryRow(ctx,
			"INSERT INTO account.registration_temp(nickname, email, password, confirmation_code) VALUES ($1, $2, $3, $4) RETURNING EXISTS(SELECT 1 FROM account.registration_temp WHERE registration_temp_no = registration_temp_no) AS result;",
			nickname, email, password, confirmCode).Scan(&resultRegistrationTempNo)

		return resultRegistrationTempNo, nil
	})

	// Block 4 - sending the result
	return result.(bool), nil
}

// AuthSignupWithCode is the resolver for the authSignupWithCode field.
func (r *mutationResolver) AuthSignupWithCode(ctx context.Context, input model.SignupWithCodeInput) (*model.AuthPayload, error) {
	// Block 1 - data validation
	nickname := strings.TrimSpace(input.Nickname)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	code := strings.TrimSpace(input.Code)

	if err := v.Validate(nickname, v.IsMinMaxLen(5, 32), v.IsContainsSpace(), v.IsNickname()); err != nil {
		return nil, errors.New("nickname: " + err.Error())
	}
	if err := v.Validate(email, v.IsMinMaxLen(6, 64), v.IsContainsSpace(), v.IsEmail()); err != nil {
		return nil, errors.New("email: " + err.Error())
	}
	if err := v.Validate(password, v.IsMinMaxLen(6, 64), v.IsContainsSpace()); err != nil {
		return nil, errors.New("password: " + err.Error())
	}
	if err := v.Validate(code, v.IsLen(6), v.IsContainsSpace(), v.IsUint64()); err != nil {
		return nil, errors.New("confirmation code from email: " + err.Error())
	}

	// Block 2 - comparing data sets
	result := execInTx(ctx, r.App.Postgres.Pool, r.App.Logrus, func(tx pgx.Tx) (interface{}, error) {
		var resultTempExists bool
		i, _ := strconv.ParseInt(code, 10, 64)
		localCode := strconv.FormatInt(i, 10)
		err := tx.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM account.registration_temp WHERE nickname = $1 AND email = $2 AND password = $3 AND confirmation_code = $4)::boolean AS temp_exists",
			nickname, email, password, localCode).Scan(&resultTempExists)
		return resultTempExists, err
	})
	existsTemp := result.(bool)
	if !existsTemp {
		return nil, errors.New("no results found")
	}

	// Block 3 - hashing password and adding a user
	base64PasswordHash, base64Salt := argon2.HashPassword(password, "", r.App.Logrus)

	result = execInTx(ctx, r.App.Postgres.Pool, r.App.Logrus, func(tx pgx.Tx) (interface{}, error) {
		var registrationTempExists uuid.UUID
		err := tx.QueryRow(ctx,
			"DELETE FROM account.registration_temp WHERE lower(nickname) = lower($1) OR email = $2 RETURNING registration_temp_no",
			nickname, email).Scan(&registrationTempExists)
		if err != nil {
			return nil, err
		}

		var resultAccountId uuid.UUID
		err = tx.QueryRow(ctx,
			"INSERT INTO account.account(type_registration) VALUES ($1) RETURNING account_id",
			"1").Scan(&resultAccountId)
		if err != nil {
			return nil, err
		}

		err = tx.QueryRow(ctx,
			"INSERT INTO account.user(account_id, email, nickname, password, salt_for_password) VALUES ($1, $2, $3, $4, $5) RETURNING account_id",
			resultAccountId, email, nickname, base64PasswordHash, base64Salt).Scan(&resultAccountId)
		return resultAccountId, err
	})
	resultAccountId := result.(uuid.UUID).String()

	// Block 4 - generating JWT
	claims := token.SetClaims(resultAccountId, r.App.Config.App.ServiceUrl)
	jwt, err := token.GenerateToken(claims, r.App.Config.App.JwtSecret)
	if err != nil {
		r.App.Logrus.NewError("the jwt not generated", err)
	}

	// Block 5 - sending the result
	return &model.AuthPayload{
		Token: jwt,
		User: &model.User{
			UUID:     resultAccountId,
			Nickname: nickname,
			Email:    email,
		},
	}, nil
}

// AuthLogin is the resolver for the authLogin field.
func (r *mutationResolver) AuthLogin(ctx context.Context, input model.LoginInput) (*model.AuthPayload, error) {
	panic(fmt.Errorf("not implemented: AuthLogin - authLogin"))
}

// AuthLogout is the resolver for the authLogout field.
func (r *mutationResolver) AuthLogout(ctx context.Context, input model.TokenInput) (bool, error) {
	panic(fmt.Errorf("not implemented: AuthLogout - authLogout"))
}

// AuthTokenValidate is the resolver for the authTokenValidate field.
func (r *mutationResolver) AuthTokenValidate(ctx context.Context, input model.TokenInput) (bool, error) {
	panic(fmt.Errorf("not implemented: AuthTokenValidate - authTokenValidate"))
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
