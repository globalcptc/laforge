package auth

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/authuser"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

// LocalLogin decodes the share session cookie and packs the session into context
func LocalLogin(client *ent.Client, laforgeConfig *utils.ServerConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		secure_cookie := false
		if laforgeConfig.UI.HttpsEnabled {
			secure_cookie = true
		}
		var loginVals login
		username := ""
		password := ""

		if err := ctx.ShouldBind(&loginVals); err != nil {
			if secure_cookie {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, true, true)
			} else {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, false, false)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err})
			return
		} else {
			username = loginVals.Username
			password = loginVals.Password
		}

		entAuthUser, err := client.AuthUser.Query().Where(
			authuser.And(
				authuser.UsernameEQ(username),
				authuser.ProviderEQ(authuser.ProviderLOCAL),
			),
		).Only(ctx)

		if err != nil {
			if secure_cookie {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, true, true)
			} else {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, false, false)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		// Compare the stored hashed password, with the hashed version of the password that was received
		if err = bcrypt.CompareHashAndPassword([]byte(entAuthUser.Password), []byte(password)); err != nil {
			if secure_cookie {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, true, true)
			} else {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, false, false)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		expiresAt := time.Now().Add(time.Minute * time.Duration(laforgeConfig.Auth.CookieTimeout)).Unix()

		claims := &Claims{
			IssuedAt: time.Now().Unix(),
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expiresAt,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			if secure_cookie {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, true, true)
			} else {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, false, false)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Error signing token"})
			return
		}

		_, err = client.Token.Create().SetAuthUser(entAuthUser).SetExpireAt(expiresAt).SetToken(tokenString).Save(ctx)
		if err != nil {
			if secure_cookie {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, true, true)
			} else {
				ctx.SetCookie("auth-cookie", "", 0, "/", laforgeConfig.Graphql.Hostname, false, false)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Error updating token"})
			return
		}

		if secure_cookie {
			ctx.SetCookie("auth-cookie", tokenString, laforgeConfig.Auth.CookieTimeout*60, "/", laforgeConfig.Graphql.Hostname, true, true)
		} else {
			ctx.SetCookie("auth-cookie", tokenString, laforgeConfig.Auth.CookieTimeout*60, "/", laforgeConfig.Graphql.Hostname, false, false)
		}

		// Hide password so no leeks
		entAuthUser.Password = ""
		ctx.JSON(200, entAuthUser)

		ctx.Next()
	}
}
