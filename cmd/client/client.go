package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "tugasbesar/api/proto/v1"
	"tugasbesar/pkg/util"

	"google.golang.org/grpc"
)

const (
	// Constants to check if flags are empty
	emptyUsername string = "username"
	emptyPassword string = "password"
)

var (
	client   pb.ChatClient
	userCred pb.UserCred
	roomList pb.RoomList
)

func main() {
	// Retrieve options from command line flag
	addressPtr := flag.String("a", "localhost:12000", "Address (IP address and port number of the server)")
	usernamePtr := flag.String("u", emptyUsername, "Username of your account")
	passwordPtr := flag.String("p", emptyPassword, "Password of your account")
	flag.Parse()

	// Set up a connection to the server.
	client = connectionSetup(addressPtr)

	// Retrieve username and password from flag
	username := *usernamePtr
	password := *passwordPtr

	// Request username and password if flag is not set
	if (username == emptyUsername) || (password == emptyPassword) {
		util.CreateLine()
		username = util.RequestInput("Username")
		password = util.RequestInput("Password")
	}
	util.CreateLine()

	// Create user credential
	userCred = pb.UserCred{Username: username, Password: util.EncryptString(password)}

	// Login to the server with credentials.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := client.Login(ctx, &userCred)
	if err != nil {
		log.Fatalln(err)
	}

	// Disconnect client when credentials is wrong
	if !r.GetIsSuccess() {
		log.Fatalln("(server) :", r.GetMessage())
	}
	fmt.Println("(server) :", r.GetMessage())
	util.CreateLine()

	// Request list of rooms available
	loadRoomList()

	for {
		// Displays list of rooms available
		showRoomList()

		// Request room name to join
		roomName := joinRoom()

		// Init thread to read messages from the server
		var wg sync.WaitGroup
		wg.Add(1)
		command := make(chan string)
		var (
			startIndex int32 = 0
			endIndex   int32 = 0
		)

		// Read messages in the room
		go routine(command, &wg, client, &roomName, &userCred, &startIndex, &endIndex)

		// Get user input to send message
		sendMessage(command, client, &roomName, &userCred)
	}
}

func connectionSetup(addressPtr *string) pb.ChatClient {
	// Connect to the server insecurely
	conn, err := grpc.Dial(*addressPtr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf(err.Error())
	}
	return pb.NewChatClient(conn)
}

func loadRoomList() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Retrieve list of rooms from the server
	stream, err := client.GetRooms(ctx, &userCred)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		// Retrieve room from stream
		room, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		// Add the room to the local room list array
		roomList.Rooms = append(roomList.GetRooms(), room)
	}
}

func showRoomList() {
	fmt.Println("Rooms : ")
	// Displays all room in local room list array
	for _, room := range roomList.GetRooms() {
		fmt.Println("->", room.GetName())
	}
	util.CreateLine()
}

func joinRoom() string {
	var roomName string
	for {
		// Request room name input
		roomName = util.RequestInput("Select Room")
		util.CreateLine()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// Join selected room
		r, err := client.JoinRoom(ctx, &pb.RoomRequest{Room: &pb.Room{Name: roomName}, Cred: &userCred})
		if err != nil {
			log.Fatalln(err)
		}
		// Valid room name
		if r.GetIsSuccess() {
			fmt.Println("(server) :", r.GetMessage())
			break
			// Invalid room name, request room name again
		} else {
			fmt.Println("(server) :", r.GetMessage())
		}
	}
	fmt.Println("(server) : Press ENTER to write a message")
	fmt.Println("(server) : Type /leave in message to leave room")
	util.CreateLine()

	return roomName
}

func getMessages(c pb.ChatClient, roomName *string, userCred *pb.UserCred, startIndex, endIndex *int32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Request messages count to be used as endIndex
	r, err := c.GetMessagesCount(ctx, userCred)
	if err != nil {
		log.Fatalln(err)
	}

	// Convert count from string to int32
	count, _ := strconv.ParseInt(r.GetMessage(), 10, 32)
	*endIndex = int32(count)

	// When startIndex has the same value as endIndex then
	// there is no new messages to show
	if startIndex != endIndex {
		// Request messages for a certain room
		stream, err := c.GetMessages(ctx, &pb.MessageRequest{Room: &pb.Room{Name: *roomName}, Cred: userCred, StartIndex: *startIndex, EndIndex: *endIndex})
		if err != nil {
			log.Fatalln(err)
		}

		for {
			// Retrieve message from stream
			message, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalln(err)
			}
			// Display message to the console
			fmt.Printf("(%s) : %s \n", message.GetCred().GetUsername(), message.GetMessage())
		}
	}
	// Mark the messages as retrieved
	*startIndex = *endIndex

	// Restart the sequence every second
	time.Sleep(time.Second)
}

func sendMessage(command chan string, c pb.ChatClient, roomName *string, userCred *pb.UserCred) {
	for {
		// Wait for ENTER
		fmt.Scanln()
		// ENTER pressed
		// Pause go routine
		command <- "Pause"
		util.CreateLine()

		// Request user input for message
		message := util.RequestInput("Your Message")
		util.CreateLine()
		// Resume go routine
		command <- "Play"

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// Send message from certain room to the server
		r, err := c.SendMessage(ctx, &pb.ClientMessage{Message: message, Room: &pb.Room{Name: *roomName}, Cred: userCred})
		if err != nil {
			log.Println(err)
		}

		// Server didn't receive the message successfully
		if !r.GetIsSuccess() {
			fmt.Println("(server) : ", r.GetMessage())
			// User leave the room
		} else if r.GetIsSuccess() && (strings.TrimSpace(message) == "/leave") {
			command <- "Stop"
			break
		}
	}
}

// Controller for the go routine
func routine(command chan string, wg *sync.WaitGroup, c pb.ChatClient, roomName *string, userCred *pb.UserCred, startIndex, endIndex *int32) {
	defer wg.Done()
	var status = "Play"
	for {
		select {
		case cmd := <-command:
			switch cmd {
			case "Stop":
				return
			case "Pause":
				status = "Pause"
			default:
				status = "Play"
			}
		default:
			if status == "Play" {
				getMessages(c, roomName, userCred, startIndex, endIndex)
			}
		}
	}
}
