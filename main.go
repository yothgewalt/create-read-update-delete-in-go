package main

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strings"
)

var (
	db  *gorm.DB
	err error
)

type Employees struct {
	gorm.Model
	Firstname string `gorm:"not null;unique" json:"firstname"`
	Lastname  string `gorm:"not null" json:"lastname"`
	Salary    uint32 `gorm:"not null" json:"salary"`
	Role      string `gorm:"not null" json:"role"`
	Age       uint8  `gorm:"not null" json:"age"`
}

func main() {
	dsn := "host=localhost user=postgres password=password dbname=database port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	db, err = connect(dsn, &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		log.Panicf("(error) failed to connect the database cause %v", err.Error())
	}

	err = db.Set("gorm:table_option", "ENGINE=InnoDB").AutoMigrate(&Employees{})
	if err != nil {
		log.Fatalf("(error) failed to auto migrate into the database cause %v", err.Error())
	}

	app := fiber.New(fiber.Config{})
	app.Use(logger.New())
	app.Use(cache.New())
	app.Use(cors.New())

	app.Post("/create", create)
	app.Get("/read", read)
	app.Get("/read/:id", readById)
	app.Put("/update/:id", updateById) /* that can update only firstname data */

	deleteFunc := app.Group("/delete", func(c *fiber.Ctx) error {
		return c.Next()
	})
	deleteFunc.Delete("/soft/:id", softDeleteById)
	deleteFunc.Delete("/hard/:id", hardDeleteById)

	err = app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}

func connect(dataSource string, gormConfig *gorm.Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(postgres.Open(dataSource), gormConfig)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func create(c *fiber.Ctx) error {
	person := new(Employees)

	if err := c.BodyParser(person); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err,
		})
	}

	person.Firstname = strings.ToLower(person.Firstname)
	person.Lastname = strings.ToLower(person.Lastname)
	person.Role = strings.ToLower(person.Role)

	if err := db.Table("employees").Select("firstname").Where("firstname = ?", person.Firstname).First(&person).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Table("employees").Model(&Employees{}).Create(person).Error; err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.Status(http.StatusCreated).JSON(person)
		}
	}

	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
		"message": "(error) that firstname has been duplicated in database, please try again and use other firstname",
	})
}

func read(c *fiber.Ctx) error {
	person := new([]Employees)

	if err := db.Table("employees").Model(&Employees{}).Find(&person).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusFound).JSON(person)
}

func readById(c *fiber.Ctx) error {
	person := new(Employees)

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := db.Table("employees").Model(&Employees{}).Where("id = ?", id).First(&person).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusFound).JSON(person)
}

func updateById(c *fiber.Ctx) error {
	type UpdateRequest struct {
		Firstname string `json:"firstname"`
	}

	person := new(UpdateRequest)

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := c.BodyParser(person); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err,
		})
	}

	if err := db.Table("employees").Model(&Employees{}).Where("id = ?", id).Update("firstname", person.Firstname).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "(changed) firstname column has been changed.",
	})
}

func softDeleteById(c *fiber.Ctx) error {
	person := new(Employees)

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := db.Table("employees").Model(&Employees{}).Where("id = ?", id).Delete(&person).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "(success) the employee data has been soft removed from database",
	})
}

func hardDeleteById(c *fiber.Ctx) error {
	person := new(Employees)

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := db.Table("employees").Model(&Employees{}).Unscoped().Where("id = ?", id).Delete(&person).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(http.StatusNoContent)
}
