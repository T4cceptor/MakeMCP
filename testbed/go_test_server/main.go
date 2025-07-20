package main

import (
	"github.com/go-fuego/fuego"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var port string = ":8120"

type User struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *gorm.DB

func main() {
	db, _ = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	db.AutoMigrate(&User{})

	s := fuego.NewServer(fuego.WithAddr("localhost" + port))

	fuego.Get(s, "/", func(c fuego.ContextNoBody) (map[string]string, error) {
		return map[string]string{"message": "Hello from Go Fuego!"}, nil
	}, fuego.OptionSummary("Welcome endpoint"), fuego.OptionDescription("Returns a welcome message"))

	fuego.Get(s, "/users", func(c fuego.ContextNoBody) ([]User, error) {
		var users []User
		db.Find(&users)
		return users, nil
	}, fuego.OptionSummary("List all users"), fuego.OptionDescription("Retrieves all users from the database"))

	fuego.Post(s, "/users", func(c fuego.ContextWithBody[User]) (*User, error) {
		user, _ := c.Body()
		db.Create(&user)
		return &user, nil
	}, fuego.OptionSummary("Create user"), fuego.OptionDescription("Creates a new user in the database"))

	fuego.Get(s, "/users/{id}", func(c fuego.ContextNoBody) (*User, error) {
		var user User
		db.First(&user, c.PathParam("id"))
		return &user, nil
	}, fuego.OptionSummary("Get user by ID"), fuego.OptionDescription("Retrieves a specific user by their ID"))

	fuego.Patch(s, "/users/{id}", func(c fuego.ContextWithBody[User]) (*User, error) {
		var user User
		db.First(&user, c.PathParam("id"))
		updates, _ := c.Body()
		db.Model(&user).Updates(updates)
		return &user, nil
	}, fuego.OptionSummary("Update user"), fuego.OptionDescription("Updates an existing user's information"))

	fuego.Delete(s, "/users/{id}", func(c fuego.ContextNoBody) (map[string]bool, error) {
		var user User
		db.First(&user, c.PathParam("id"))
		db.Delete(&user)
		return map[string]bool{"ok": true}, nil
	}, fuego.OptionSummary("Delete user"), fuego.OptionDescription("Removes a user from the database"))

	// Check if user exists, create one if not
	var users []User
	db.Find(&users)
	if len(users) == 0 {
		db.Create(&User{
			Name:  "Test user",
			Email: "test@test.com",
		})
	}

	s.Run()
}
