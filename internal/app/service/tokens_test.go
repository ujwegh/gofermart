package service

import (
	"strings"
	"testing"
)

func TestTokenServiceImpl_GetUserLogin(t *testing.T) {
	validSecretKey := "super-duper-secret"
	differentSecretKey := "different-secret-key"
	validTokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJnb3BoZXJtYXJ0Iiwic3ViIjoiYXV0aCB0b2tlbiIsImV4cCI6MTczNDg2MTE1NSwiaWF0IjoxNzAzMzI1MTU1LCJVc2VyTG9naW4iOiJkaW5DVkVkIn0.pGy52Pdxynv0c94ZnMKx5FvC_PvIJSjP92BJhB9NKFw"
	invalidTokenString := "invalid-token"
	emptyLoginTokenString := ""
	expiredTokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJnb3BoZXJtYXJ0Iiwic3ViIjoiYXV0aCB0b2tlbiIsImV4cCI6MTcwMjkyODEzNywiaWF0IjoxNzAyOTI4MTM2LCJVc2VyTG9naW4iOiJkaW5DVkVkIn0.RgZNKAbl3y563RUWXy7ohAq2TIVTtbWhjyeEO2b2KPw"
	differentKeyTokenString := "different-key-token"

	type fields struct {
		secretKey string
	}
	type args struct {
		tokenString string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		want        string
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Valid Token",
			fields: fields{
				secretKey: validSecretKey,
			},
			args: args{
				tokenString: validTokenString,
			},
			want:        "dinCVEd",
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "Invalid Token",
			fields: fields{
				secretKey: validSecretKey,
			},
			args: args{
				tokenString: invalidTokenString,
			},
			want:        "",
			wantErr:     true,
			expectedErr: "token error: failed to parse token: token contains an invalid number of segments",
		},
		{
			name: "Empty User Login in Token",
			fields: fields{
				secretKey: validSecretKey,
			},
			args: args{
				tokenString: emptyLoginTokenString,
			},
			want:        "",
			wantErr:     true,
			expectedErr: "token error: failed to parse token: token contains an invalid number of segments",
		},
		{
			name: "Expired Token",
			fields: fields{
				secretKey: validSecretKey,
			},
			args: args{
				tokenString: expiredTokenString,
			},
			want:        "",
			wantErr:     true,
			expectedErr: "token error: failed to parse token: token is expired by",
		},
		{
			name: "Token Signed With Different Key",
			fields: fields{
				secretKey: validSecretKey,
			},
			args: args{
				tokenString: differentKeyTokenString,
			},
			want:        "",
			wantErr:     true,
			expectedErr: "token error: failed to parse token: token contains an invalid number of segments",
		},
		{
			name: "Token With Unexpected Signing Method",
			fields: fields{
				secretKey: differentSecretKey,
			},
			args: args{
				tokenString: validTokenString,
			},
			want:        "",
			wantErr:     true,
			expectedErr: "token error: failed to parse token: signature is invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := TokenServiceImpl{
				secretKey: tt.fields.secretKey,
			}
			got, err := ts.GetUserLogin(tt.args.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserLogin() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr && !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("GetUserLogin() unexpected error message = %v, want %v", err, tt.expectedErr)
			}
			if got != tt.want {
				t.Errorf("GetUserLogin() got = %v, want %v", got, tt.want)
			}
		})
	}
}
