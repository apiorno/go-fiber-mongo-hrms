package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

var mg MongoInstance

func Connect() error {
	mongoURI := fmt.Sprintf("mongodb://%v:%v@%v:%v/%v", os.Getenv("USER"), os.Getenv("PASS"), os.Getenv("HOST"), os.Getenv("PORT"), os.Getenv("DB"))
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(os.Getenv("DB"))
	if err != nil {
		return err
	}
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil

}
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error getting env, not comming through %v", err)
	}
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()
	app.Get("/employees", func(ctx *fiber.Ctx) {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(ctx.Context(), query)
		if err != nil {
			ctx.Status(500).SendString(err.Error())
			return
		}
		var employees []Employee = make([]Employee, 0)
		if err := cursor.All(ctx.Context(), &employees); err != nil {
			ctx.Status(500).SendString(err.Error())
			return
		}
		ctx.JSON(employees)
	})

	app.Post("/employees", func(ctx *fiber.Ctx) {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)
		if err := ctx.BodyParser(employee); err != nil {
			ctx.Status(400).SendString(err.Error())
			return
		}
		employee.ID = ""

		result, err := collection.InsertOne(ctx.Context(), employee)
		if err != nil {
			ctx.Status(500).SendString(err.Error())
			return
		}
		filter := bson.D{{Key: "_id", Value: result.InsertedID}}
		cratedRecord := collection.FindOne(ctx.Context(), filter)

		createdEmployee := &Employee{}
		err = cratedRecord.Decode(createdEmployee)
		if err != nil {
			ctx.Status(500).SendString(err.Error())
			return
		}

		ctx.Status(201).JSON(createdEmployee)

	})

	app.Put("/employees/:id", func(ctx *fiber.Ctx) {
		id := ctx.Params("id")
		employeeId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			ctx.Status(400).SendString(err.Error())
			return
		}
		employee := new(Employee)
		if err := ctx.BodyParser(employee); err != nil {
			ctx.Status(400).SendString(err.Error())
		}
		query := bson.D{{Key: "_id", Value: employeeId}}
		update := bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: employee.Name},
			{Key: "age", Value: employee.Age},
			{Key: "salary", Value: employee.Salary},
		}}}
		err = mg.Db.Collection("employees").FindOneAndUpdate(ctx.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				ctx.SendStatus(400)
				return
			}
			ctx.SendStatus(500)
			return
		}
		employee.ID = id
		ctx.Status(200).JSON(employee)

	})
	app.Delete("/employees/:id", func(ctx *fiber.Ctx) {
		id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
		if err != nil {
			ctx.SendStatus(400)
		}
		query := bson.D{{Key: "_id", Value: id}}
		result, err := mg.Db.Collection("employees").DeleteOne(ctx.Context(), &query)

		if err != nil {
			ctx.SendStatus(500)
			return
		}
		if result.DeletedCount < 1 {
			ctx.SendStatus(404)
			return
		}
		ctx.Status(200).JSON("Record deleted")

	})
	log.Fatal(app.Listen(":3000"))
}
