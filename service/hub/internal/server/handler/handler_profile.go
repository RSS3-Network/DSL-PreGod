package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/naturalselectionlabs/pregod/common/database"
	"github.com/naturalselectionlabs/pregod/common/database/model/social"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/worker/crossbell"
	"github.com/naturalselectionlabs/pregod/common/worker/ens"
	"github.com/naturalselectionlabs/pregod/common/worker/lens"
	lop "github.com/samber/lo/parallel"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm/clause"
)

var ProfilePlatformList = []string{
	protocol.PlatformEns,
	protocol.PlatformLens,
	protocol.PlatformCrossbell,
}

// GetProfilesFunc supported filter:
// - address
// - network
// - platform
func (h *Handler) GetProfilesFunc(c echo.Context) error {
	go h.apiReport(GetProfiles, c)
	tracer := otel.Tracer("GetProfilesFunc")
	ctx, httpSnap := tracer.Start(c.Request().Context(), "http")

	defer httpSnap.End()

	request := GetRequest{}

	if err := c.Bind(&request); err != nil {
		return BadRequest(c)
	}

	if err := c.Validate(&request); err != nil {
		return err
	}

	go h.filterReport(GetProfiles, request)

	profileList, err := h.getProfiles(ctx, request)
	if err != nil {
		return InternalError(c)
	}

	total := int64(len(profileList))
	return c.JSON(http.StatusOK, &Response{
		Total:  &total,
		Result: profileList,
	})
}

func (h *Handler) BatchGetProfilesFunc(c echo.Context) error {
	go h.apiReport(PostProfiles, c)
	tracer := otel.Tracer("BatchGetProfilesFunc")
	ctx, httpSnap := tracer.Start(c.Request().Context(), "http")

	defer httpSnap.End()

	request := BatchGetProfilesRequest{}

	if err := c.Bind(&request); err != nil {
		return BadRequest(c)
	}

	if err := c.Validate(&request); err != nil {
		return err
	}

	go h.filterReport(PostProfiles, request)

	if len(request.Address) > DefaultLimit {
		request.Address = request.Address[:DefaultLimit]
	}

	profileList, err := h.batchGetProfiles(ctx, request)
	if err != nil {
		return InternalError(c)
	}

	total := int64(len(profileList))
	return c.JSON(http.StatusOK, &Response{
		Total:  &total,
		Result: profileList,
	})
}

func (h *Handler) getProfiles(c context.Context, request GetRequest) ([]social.Profile, error) {
	m := make(map[string]social.Profile)
	result := make([]social.Profile, 0)

	profiles, _ := h.getProfilesDatabase(c, request)

	for _, profile := range profiles {
		m[profile.Platform] = profile
		result = append(result, profile)
	}

	lop.ForEach(ProfilePlatformList, func(platform string, i int) {
		if profile, ok := m[platform]; ok {
			go func() {
				if profile.UpdatedAt.After(time.Now().Add(-2 * time.Minute)) {
					return
				}

				if _, err := h.getProfilesFromPlatform(c, platform, request.Address); err != nil {
					logrus.Errorf("[profile] getProfilesFromPlatform error, %v", err)
				}
			}()

			return
		}

		list, err := h.getProfilesFromPlatform(c, platform, request.Address)
		if err == nil && len(list) > 0 {
			result = append(result, list...)
		}
	})

	return result, nil
}

func (h *Handler) batchGetProfiles(c context.Context, request BatchGetProfilesRequest) ([]social.Profile, error) {
	m := make(map[string]social.Profile)
	result := make([]social.Profile, 0)

	profiles, _ := h.batchGetProfilesDatabase(c, request)

	for _, profile := range profiles {
		key := fmt.Sprintf("%v:%v", profile.Platform, profile.Address)
		m[key] = profile
		result = append(result, profile)
	}

	lop.ForEach(request.Address, func(address string, i int) {
		lop.ForEach(ProfilePlatformList, func(platform string, i int) {
			key := fmt.Sprintf("%v:%v", platform, address)
			if profile, ok := m[key]; ok {
				go func() {
					if profile.UpdatedAt.After(time.Now().Add(-2 * time.Minute)) {
						return
					}

					if _, err := h.getProfilesFromPlatform(c, platform, address); err != nil {
						return
					}
				}()

				return
			}

			list, err := h.getProfilesFromPlatform(c, platform, address)
			if err == nil && len(list) > 0 {
				result = append(result, list...)
			}
		})
	})

	return result, nil
}

func (h *Handler) getProfilesFromPlatform(c context.Context, platform, address string) ([]social.Profile, error) {
	var profile *social.Profile
	var profiles []social.Profile
	var err error

	switch platform {
	case protocol.PlatformEns:
		ensClient := ens.New()
		profile, err = ensClient.GetProfile(address)
	case protocol.PlatformLens:
		lensClient := lens.New()
		profile, err = lensClient.GetProfile(address)
	case protocol.PlatformCrossbell:
		csbClient := crossbell.New()
		profiles, err = csbClient.GetProfile(address)
	}

	if err != nil {
		logrus.Errorf("[profile] getProfilesFromPlatform error, %v", err)
		return nil, err
	}

	if profile != nil {
		profiles = append(profiles, *profile)
	}

	if len(profiles) == 0 {
		return nil, nil
	}

	go h.upsertProfiles(c, profiles)

	return profiles, err
}

// getProfiles get profile data from database
func (h *Handler) getProfilesDatabase(c context.Context, request GetRequest) ([]social.Profile, error) {
	tracer := otel.Tracer("getProfiles")
	_, postgresSnap := tracer.Start(c, "postgres")

	defer postgresSnap.End()

	dbResult := make([]social.Profile, 0)

	sql := database.Global().Model(&social.Profile{}).Where("LOWER(address) = ? ",
		strings.ToLower(request.Address),
	)

	if len(request.Network) > 0 {
		for i, v := range request.Network {
			request.Network[i] = strings.ToLower(v)
		}
		sql = sql.Where("network IN ?", request.Network)
	}

	if len(request.Platform) > 0 {
		for i, v := range request.Platform {
			request.Platform[i] = strings.ToLower(v)
		}
		sql = sql.Where("platform IN ?", request.Platform)
	}

	sql = sql.Limit(DefaultLimit)

	if err := sql.Find(&dbResult).Error; err != nil {
		logrus.Errorf("[profile] getProfilesDatabase error, %v", err)

		return nil, err
	}

	return dbResult, nil
}

// batchGetProfiles get profile data from database
func (h *Handler) batchGetProfilesDatabase(c context.Context, request BatchGetProfilesRequest) ([]social.Profile, error) {
	tracer := otel.Tracer("batchGetProfiles")
	_, postgresSnap := tracer.Start(c, "postgres")

	defer postgresSnap.End()

	dbResult := make([]social.Profile, 0)

	if len(request.Address) > DefaultLimit {
		request.Address = request.Address[:DefaultLimit]
	}

	for i, v := range request.Address {
		request.Address[i] = strings.ToLower(v)
	}

	sql := database.Global().Model(&social.Profile{}).Where("LOWER(address) IN ? ", request.Address)

	if len(request.Network) > 0 {
		for i, v := range request.Network {
			request.Network[i] = strings.ToLower(v)
		}
		sql = sql.Where("network IN ?", request.Network)
	}

	if len(request.Platform) > 0 {
		for i, v := range request.Platform {
			request.Platform[i] = strings.ToLower(v)
		}
		sql = sql.Where("platform IN ?", request.Platform)
	}

	if err := sql.Find(&dbResult).Error; err != nil {
		return nil, err
	}

	return dbResult, nil
}

func (h *Handler) upsertProfiles(c context.Context, profiles []social.Profile) {
	err := database.Global().Model(&social.Profile{}).
		Clauses(clause.OnConflict{
			UpdateAll: true,
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).
		Create(profiles).Error
	if err != nil {
		logrus.Errorf("[profile] upsertProfiles error, %v", err)
	}
}
