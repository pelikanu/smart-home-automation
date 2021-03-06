package authentication

import (
	"crypto/rsa"
	"os"
	"bufio"
	"encoding/pem"
	"crypto/x509"
	"smart-home-automation/core/redis"
	"github.com/dgrijalva/jwt-go"
	"time"
	"golang.org/x/crypto/bcrypt"
	"smart-home-automation/models"
	"github.com/pborman/uuid"
	"log"
	"smart-home-automation/settings"
	"fmt"
)

type JWTAuthenticationBackend struct {
	privateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

const (
	tokenDuration = 30
	expireOffset  = 3600
)

var authBackendInstance *JWTAuthenticationBackend = nil

func InitJWTAuthenticationBackend() *JWTAuthenticationBackend {

	if authBackendInstance == nil {
		authBackendInstance = &JWTAuthenticationBackend{
			privateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}

	return authBackendInstance
}

func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open("settings/keys/public_key.pem")
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := publicKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)
	data, _ := pem.Decode([]byte(pembytes))

	publicKeyFile.Close()

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)
	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}

	return rsaPub

}

func getPrivateKey() *rsa.PrivateKey {

	privateKeyFile, err := os.Open("settings/keys/private_key.pem")
	if err != nil {
		log.Fatal(err)
	}

	pemfileinfo, _ := privateKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	privateKeyFile.Close()

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)

	if err != nil {
		log.Fatal(err)
	}

	return privateKeyImported
}

func (backend *JWTAuthenticationBackend) Logout(tokenString string, token *jwt.Token) error {
	redisConn := redis.Connect()
	return redisConn.SetValue(tokenString, tokenString, backend.getTokenRemainingValidity(token.Claims.(jwt.MapClaims)["exp"]))
}

func (backend *JWTAuthenticationBackend) getTokenRemainingValidity(timestamp interface{}) int {
	if validity, ok := timestamp.(float64); ok {
		tm := time.Unix(int64(validity), 0)
		remainer := tm.Sub(time.Now())
		if remainer > 0 {
			return int(remainer.Seconds() + expireOffset)
		}
	}
	return expireOffset
}

func (backend *JWTAuthenticationBackend) Authenticate(user *models.User) bool {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testing"), 10)

	testUser := models.User{
		UUID:     uuid.New(),
		Username: "haku",
		Password: string(hashedPassword),
	}

	fmt.Println(testUser.UUID)

	return user.Username == testUser.Username && bcrypt.CompareHashAndPassword([]byte(testUser.Password), []byte(user.Password)) == nil
}

func (backend *JWTAuthenticationBackend) GenerateToken(userUUID string) (string, error) {
	token :=jwt.New(jwt.SigningMethodRS512)


	token.Claims = jwt.MapClaims{
		"admin":"true",
		"tst": "xx",
		"exp": time.Now().Add(time.Second * time.Duration(settings.Settings{}.JWTExpirationDelta)).Unix(),
		"iat": time.Now().Unix(),
		"sub": userUUID,
	}

	//create a rsa 256 signer


	tokenString,err:=token.SignedString(backend.privateKey)

	if err!=nil {
		panic(err)
		return "", err
	}



	return tokenString, nil
}

func CheckToken(tokenStr string) (string, error) {

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// check token signing method etc
		return getPublicKey(), nil
	})

	if err!=nil{
		fmt.Println(err)
	}

	log.Println(token)
	// do something with decoded claims
	for key, val := range claims {
		fmt.Printf("Key: %v, value: %v\n", key, val)
	}

	return "x", err
}

