package models

import (
	"encoding/base64"
	"errors"
	"es_middleware/config"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/opentreehole/go-common"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"
	"strconv"
	"strings"
	"time"
)

type User struct {
	/// base info
	ID int `json:"id" gorm:"primaryKey"`

	Config UserConfig `json:"config" gorm:"serializer:json;not null;default:\"{}\""`

	BanDivision map[int]*time.Time `json:"-" gorm:"serializer:json;not null;default:\"{}\""`

	OffenceCount int `json:"-" gorm:"not null;default:0"`

	/// association fields, should add foreign key

	// favorite holes of the user
	UserFavoriteHoles Holes `json:"-" gorm:"many2many:user_favorite"`

	// holes owned by the user
	UserHoles Holes `json:"-"`

	// floors owned by the user
	UserFloors Floors `json:"-"`

	// reports made by the user; a user has many report
	UserReports Reports `json:"-"`

	// floors liked by the user
	UserLikedFloors Floors `json:"-" gorm:"many2many:floor_like"`

	// floors disliked by the user
	UserDislikedFloors Floors `json:"-" gorm:"many2many:floor_dislike"`

	// floor history made by the user
	UserFloorHistory FloorHistorySlice `json:"-"`

	// user punishments on division
	UserPunishments Punishments `json:"-"`

	// punishments made by this user
	UserMakePunishments Punishments `json:"-" gorm:"foreignKey:MadeBy"`

	/// dynamically generated field

	UserID int `json:"user_id" gorm:"-:all"`

	Permission struct {
		// 管理员权限到期时间
		Admin time.Time `json:"admin"`
		// key: division_id value: 对应分区禁言解除时间
		Silent       map[int]*time.Time `json:"silent"`
		OffenseCount int                `json:"offense_count"`
	} `json:"permission" gorm:"-:all"`

	// get from jwt
	IsAdmin    bool      `json:"is_admin" gorm:"-:all"`
	JoinedTime time.Time `json:"joined_time" gorm:"-:all"`
	Nickname   string    `json:"nickname" gorm:"-:all"`
}

type Users []*User

type UserConfig struct {
	// used when notify
	Notify []string `json:"notify"`

	// 对折叠内容的处理
	// fold 折叠, hide 隐藏, show 展示
	ShowFolded string `json:"show_folded"`
}

var defaultUserConfig = UserConfig{
	Notify:     []string{"mention", "favorite", "report"},
	ShowFolded: "fold",
}

func (user *User) GetID() int {
	return user.ID
}

func (user *User) AfterCreate(_ *gorm.DB) error {
	user.UserID = user.ID
	return nil
}

func (user *User) AfterFind(_ *gorm.DB) error {
	user.UserID = user.ID
	return nil
}

// parseJWT extracts and parse token
func (user *User) parseJWT(token string) error {
	if len(token) < 7 {
		return errors.New("bearer token required")
	}

	payloads := strings.SplitN(token[7:], ".", 3) // extract "Bearer "
	if len(payloads) < 3 {
		return errors.New("jwt token required")
	}

	// jwt encoding ignores padding, so RawStdEncoding should be used instead of StdEncoding
	payloadBytes, err := base64.RawStdEncoding.DecodeString(payloads[1]) // the middle one is payload
	if err != nil {
		return err
	}

	err = json.Unmarshal(payloadBytes, user)
	if err != nil {
		return err
	}

	return nil
}

var (
	maxTime time.Time
	minTime time.Time
)

func init() {
	var err error
	maxTime, err = time.Parse(time.RFC3339, "9999-01-01T00:00:00+00:00")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	minTime = time.Unix(0, 0)
}

func GetUser(c *fiber.Ctx) (*User, error) {
	user := &User{
		BanDivision: make(map[int]*time.Time),
	}
	if config.Config.Mode == "dev" || config.Config.Mode == "test" {
		user.ID = 1
		user.IsAdmin = true
		return user, nil
	}

	if c.Locals("user") != nil {
		return c.Locals("user").(*User), nil
	}

	// get id
	userID, err := GetUserID(c)
	if err != nil {
		return nil, err
	}

	// parse JWT first
	tokenString := c.Get("Authorization")
	if tokenString == "" { // token can be in either header or cookie
		tokenString = c.Cookies("access")
	}
	err = user.parseJWT(tokenString)
	if err != nil {
		return nil, common.Unauthorized(err.Error())
	}

	// load user from database in transaction
	err = user.LoadUserByID(userID)

	if user.IsAdmin {
		user.Permission.Admin = maxTime
	} else {
		user.Permission.Admin = minTime
	}
	user.Permission.Silent = user.BanDivision
	user.Permission.OffenseCount = user.OffenceCount

	// save user in c.Locals
	c.Locals("user", user)

	return user, err
}

func GetUserID(c *fiber.Ctx) (int, error) {
	if config.Config.Mode == "dev" || config.Config.Mode == "test" {
		return 1, nil
	}

	id, err := strconv.Atoi(c.Get("X-Consumer-Username"))
	if err != nil {
		return 0, common.Unauthorized("Unauthorized")
	}

	return id, nil
}

func (user *User) LoadUserByID(userID int) error {
	return DB.Clauses(dbresolver.Write).Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("UserPunishments").Clauses(clause.Locking{Strength: "UPDATE"}).Take(&user, userID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// insert user if not found
				user.ID = userID
				user.Config = defaultUserConfig
				err = tx.Create(&user).Error
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// check permission
		modified := false
		for divisionID := range user.BanDivision {
			// get the latest punishments in divisionID
			var latestPunishment *Punishment
			for i := range user.UserPunishments {
				if user.UserPunishments[i].DivisionID == divisionID {
					latestPunishment = user.UserPunishments[i]
				}
			}

			if latestPunishment == nil || latestPunishment.EndTime.Before(time.Now()) {
				delete(user.BanDivision, divisionID)
				modified = true
			}
		}

		if modified {
			err = tx.Select("BanDivision").Save(&user).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (user *User) BanDivisionMessage(divisionID int) string {
	if user.BanDivision[divisionID] == nil {
		return fmt.Sprintf("您在此板块已被禁言")
	} else {
		return fmt.Sprintf("您在此板块已被禁言，解封时间：%s", user.BanDivision[divisionID].Format("2006-01-02 15:04:05"))
	}
}
