package models

import (
	"es_middleware/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Division struct {
	/// saved fields
	ID        int       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"time_created" gorm:"not null"`
	UpdatedAt time.Time `json:"time_updated" gorm:"not null"`

	/// base info
	Name        string `json:"name" gorm:"unique;size:10"`
	Description string `json:"description" gorm:"size:64"`

	// pinned holes in given order
	Pinned []int `json:"-" gorm:"serializer:json;size:100;not null;default:\"[]\""`

	/// association fields, should add foreign key

	// return pinned hole to frontend
	Holes Holes `json:"pinned"`

	/// generated field
	DivisionID int `json:"division_id" gorm:"-:all"`
}

func (division *Division) GetID() int {
	return division.ID
}

type Divisions []*Division

func (divisions Divisions) Preprocess(c *fiber.Ctx) error {
	for _, division := range divisions {
		err := division.Preprocess(c)
		if err != nil {
			return err
		}
	}
	return utils.SetCache("divisions", divisions, 0)
}

func (division *Division) Preprocess(c *fiber.Ctx) error {
	var pinned = division.Pinned
	if len(pinned) == 0 {
		division.Holes = Holes{}
		return nil
	}
	var holes Holes
	DB.Find(&holes, pinned)
	holes = utils.OrderInGivenOrder(holes, pinned)
	err := holes.Preprocess(c)
	if err != nil {
		return err
	}
	division.Holes = holes
	return nil
}

func (division *Division) AfterFind(_ *gorm.DB) (err error) {
	division.DivisionID = division.ID
	return nil
}

func (division *Division) AfterCreate(_ *gorm.DB) (err error) {
	division.DivisionID = division.ID
	return nil
}
