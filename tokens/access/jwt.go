package access

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"os"
	"time"
)

type Producer struct {
	key *rsa.PrivateKey
}

func NewProducer(privateKeyPath string) (*Producer, error) {
	data, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		return nil, err
	}

	return &Producer{key: key}, nil
}

func (p *Producer) Produce(payload Payload, ttl time.Duration) (string, error) {

	currentTime := time.Now().UTC()

	claims := make(jwt.MapClaims)
	claims[ClaimUserId] = payload.UserId.String()
	claims[ClaimEmail] = payload.Email
	if payload.Nickname != nil {
		claims[ClaimNickname] = *payload.Nickname
	}
	claims[ClaimRole] = payload.Role
	claims[ClaimSubscriptionPlan] = payload.SubscriptionPlan
	claims[ClaimExpiration] = currentTime.Add(ttl).Unix()
	claims[ClaimIssuedAtTime] = currentTime.Unix()
	claims[ClaimNotBefore] = currentTime.Unix()

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(p.key)
	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return token, nil
}

type Parser struct {
	Key *rsa.PublicKey
}

func NewParser(publicKeyPath string) (*Parser, error) {
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, err
	}

	return &Parser{Key: key}, nil
}

func NewParserByKey(data []byte) (*Parser, error) {
	key, err := x509.ParsePKCS1PublicKey(data)
	if err != nil {
		key, err = jwt.ParseRSAPublicKeyFromPEM(data)
	}
	if err != nil {
		return nil, err
	}

	return &Parser{Key: key}, nil
}

func (p *Parser) Parse(token string) (Payload, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (i interface{}, err error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid signing method: %v", token.Header["alg"])
		}

		return p.Key, nil
	})
	if err != nil {
		return Payload{}, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return Payload{}, fmt.Errorf("invalid token")
	}

	userId, err := uuid.Parse(claims[ClaimUserId].(string))
	if err != nil {
		return Payload{}, err
	}
	var nicknamePtr *string = nil
	if nickname := claims[ClaimNickname].(string); len(nickname) > 0 {
		nicknamePtr = &nickname
	}

	return Payload{
		UserId:           userId,
		Email:            claims[ClaimEmail].(string),
		Nickname:         nicknamePtr,
		Role:             claims[ClaimRole].(string),
		SubscriptionPlan: claims[ClaimSubscriptionPlan].(string),
	}, nil
}
