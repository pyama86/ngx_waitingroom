package waitingroom

import (
	"fmt"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/securecookie"
	"github.com/labstack/echo/v4"
)

type QueueConfirmation struct {
	sc          *securecookie.SecureCookie
	cache       *Cache
	redisClient *redis.Client
	config      *Config
}

func NewQueueConfirmation(
	sc *securecookie.SecureCookie,
	config *Config,
	redisClient *redis.Client,
	cache *Cache,
) *QueueConfirmation {
	return &QueueConfirmation{
		sc:          sc,
		config:      config,
		redisClient: redisClient,
		cache:       cache,
	}
}

const paramDomainKey = "domain"

func (p *QueueConfirmation) Do(c echo.Context) error {
	client, err := NewClientByContext(c, p.sc)
	if err != nil {
		return NewError(http.StatusInternalServerError, err, " can't build info")
	}
	site := NewSite(c.Request().Context(), c.Param(paramDomainKey), p.config, p.redisClient, p.cache)
	c.Logger().Debugf("domain %s request client info: %#v", site.domain, client)
	ok, err := site.isInWhitelist()
	if err != nil {
		return NewError(http.StatusInternalServerError, err, " can't get whitelist")
	}

	if ok {
		return c.JSON(http.StatusOK, "site is in whitelist")
	}
	if c.Param("enable") != "" {
		if err := site.EnableQueue(); err != nil {
			return NewError(http.StatusInternalServerError, err, " can't enable queue")
		}
	}

	if site.isPermittedClient(client) {
		return c.JSON(http.StatusOK, "permitted client")
	}

	clientSerialNumber, err := client.fillSerialNumber(site)
	if err != nil {
		return NewError(http.StatusInternalServerError, err, " can't get serial no")
	}

	if clientSerialNumber != 0 {
		ok, err := site.isClientPermit(client)
		if err != nil {
			return NewError(http.StatusInternalServerError, err, " can't jude permit access")
		}
		if ok {
			return c.JSON(http.StatusOK, "permitted connection")
		}
	}

	if err := client.saveToCookie(c, p.config); err != nil {
		return NewError(http.StatusInternalServerError, err, "can't save client info")
	}

	cp := int64(0)
	if client.SerialNumber != 0 {
		lcp, err := site.currentPermitedNumber(true)
		if err != nil {
			return NewError(http.StatusInternalServerError, err, "can't get current no")
		}
		cp = lcp
	}

	return c.String(http.StatusTooManyRequests, fmt.Sprintf(`{"serial_no": %d, "permitted_no": %d }`,
		client.SerialNumber, cp))
}
