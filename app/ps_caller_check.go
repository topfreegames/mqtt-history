package app

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/labstack/echo"
	"github.com/spf13/viper"

	"github.com/topfreegames/mqtt-history/logger"
)

const (
	psAuthModeOff     = "off"
	psAuthModeLog     = "log"
	psAuthModeEnforce = "enforce"

	psCallerResultCredential = "credential"
	psCallerResultAddress    = "address"
	psCallerResultRefused    = "refused"

	psCallerUnknown = "unknown"
)

type psCredential struct {
	caller   string
	user     string
	password string
}

type psAddress struct {
	caller  string
	address string
}

// PSAuth is the caller check configuration of the player support route.
type PSAuth struct {
	Mode        string
	credentials []psCredential
	addresses   []psAddress
}

func loadPSAuth(config *viper.Viper) (*PSAuth, error) {
	mode := config.GetString("ps.auth.mode")
	switch mode {
	case psAuthModeOff:
		return &PSAuth{Mode: mode}, nil
	case psAuthModeLog, psAuthModeEnforce:
	default:
		return nil, fmt.Errorf("ps.auth.mode must be off, log or enforce, got %q", mode)
	}

	psAuth := &PSAuth{Mode: mode}

	for _, caller := range splitPSAuthList(config.GetString("ps.auth.callers")) {
		if caller == "" {
			return nil, fmt.Errorf("ps.auth.callers has an empty entry")
		}
		value := config.GetString("ps.auth.credentials." + caller)
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("ps.auth.credentials.%s must be user:password", caller)
		}
		psAuth.credentials = append(psAuth.credentials, psCredential{
			caller:   caller,
			user:     parts[0],
			password: parts[1],
		})
	}

	for _, entry := range splitPSAuthList(config.GetString("ps.auth.addresses")) {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || parts[0] == "" || net.ParseIP(parts[1]) == nil {
			return nil, fmt.Errorf("ps.auth.addresses entry %q must be name=ip", entry)
		}
		psAuth.addresses = append(psAuth.addresses, psAddress{caller: parts[0], address: parts[1]})
	}

	if len(psAuth.credentials) == 0 && len(psAuth.addresses) == 0 {
		return nil, fmt.Errorf("ps.auth.mode %s needs at least one caller or address", mode)
	}

	return psAuth, nil
}

func splitPSAuthList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	entries := strings.Split(value, ",")
	for i, entry := range entries {
		entries[i] = strings.TrimSpace(entry)
	}
	return entries
}

func parseBasicAuth(header string) (user, password string, ok bool) {
	const prefix = "basic "
	if len(header) < len(prefix) || strings.ToLower(header[:len(prefix)]) != prefix {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func checkPSCaller(app *App, c echo.Context) (caller string, result string) {
	header := c.Request().Header()

	if user, password, ok := parseBasicAuth(header.Get("Authorization")); ok {
		for _, credential := range app.PSAuth.credentials {
			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(credential.user))
			passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(credential.password))
			if userMatch&passwordMatch == 1 {
				return credential.caller, psCallerResultCredential
			}
		}
	}

	if realIP := header.Get("X-Real-IP"); realIP != "" {
		for _, address := range app.PSAuth.addresses {
			if realIP == address.address {
				return address.caller, psCallerResultAddress
			}
		}
	}

	return psCallerUnknown, psCallerResultRefused
}

func psCallerAllowed(app *App, c echo.Context) bool {
	if app.PSAuth.Mode == psAuthModeOff {
		return true
	}

	caller, result := checkPSCaller(app, c)
	PSCallerCheckTotal.WithLabelValues(result, caller).Inc()
	if result != psCallerResultRefused {
		return true
	}

	request := c.Request()
	presentedUser := "none"
	if user, _, ok := parseBasicAuth(request.Header().Get("Authorization")); ok {
		presentedUser = user
	}
	logger.Logger.Warningf(
		"ps caller check refused a request: mode=%s host=%q x_real_ip=%q user_agent=%q user=%q",
		app.PSAuth.Mode, request.Host(), request.Header().Get("X-Real-IP"), request.UserAgent(), presentedUser)

	return app.PSAuth.Mode != psAuthModeEnforce
}
