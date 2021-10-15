//go:generate protoc -I ../../api/proto/v1 --go_out=plugins=grpc:../../api/proto/v1 ../../api/proto/v1/chat.proto

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"strconv"

	pb "tugasbesar/api/proto/v1"
	"tugasbesar/pkg/util"

	"google.golang.org/grpc"
)

var (
	userList    pb.UserList
	roomList    pb.RoomList
	chatHistory pb.ChatHistory
)

// server is used to implement chat.ChatServer.
type server struct {
	pb.UnimplementedChatServer
}

// Login implementation allows client to sign in to the chat service
func (s *server) Login(ctx context.Context, userCred *pb.UserCred) (*pb.Reply, error) {
	log.Printf("%-20s : %s (%s)", "Login attempt", userCred.GetUsername(), userCred.GetPassword())
	if checkCred(&userList, userCred) {
		log.Printf("%-20s : %s", "Logged in", userCred.GetUsername())
		return &pb.Reply{IsSuccess: true, Message: "Hello " + userCred.GetUsername()}, nil
	}
	return &pb.Reply{IsSuccess: false, Message: "Wrong Username or Password"}, nil
}

// GetRooms implementation allows client to retrieve all available chat rooms in the server
func (s *server) GetRooms(user *pb.UserCred, stream pb.Chat_GetRoomsServer) error {
	if checkCred(&userList, user) {
		for _, room := range roomList.GetRooms() {
			if err := stream.Send(room); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

// JoinRoom implementation allows client to join a chat room in the server
func (s *server) JoinRoom(ctx context.Context, request *pb.RoomRequest) (*pb.Reply, error) {
	if checkCred(&userList, request.GetCred()) {
		if checkRoom(&roomList, request.GetRoom()) {
			chatHistory.Messages = append(chatHistory.GetMessages(), &pb.ClientMessage{
				Message: request.GetCred().GetUsername() + " has joined the room",
				Room:    &pb.Room{Name: request.GetRoom().GetName()},
				Cred:    &pb.UserCred{Username: "server", Password: ""},
			})
			log.Printf("%-20s : %s (%s)", "Joined to", request.GetRoom().GetName(), request.GetCred().GetUsername())
			return &pb.Reply{IsSuccess: true, Message: "You are now in " + request.GetRoom().GetName()}, nil
		}
		return &pb.Reply{IsSuccess: false, Message: "No such room"}, nil
	}
	return &pb.Reply{IsSuccess: false, Message: "Wrong Username or Password"}, nil
}

// SendMessage implementation allows client to send message to a certain room to the server
func (s *server) SendMessage(ctx context.Context, message *pb.ClientMessage) (*pb.Reply, error) {
	if checkCred(&userList, message.GetCred()) {
		if checkRoom(&roomList, message.GetRoom()) {
			if message.GetMessage() == "/leave" {
				message.Message = message.GetCred().GetUsername() + " has left the room"
				message.Cred.Username = "server"
			}
			message.Cred.Password = ""
			chatHistory.Messages = append(chatHistory.GetMessages(), message)
			log.Printf("%-20s : %s (%s)", "Received message to", message.GetRoom().GetName(), message.GetCred().GetUsername())
			return &pb.Reply{IsSuccess: true, Message: "Message from " + message.GetCred().GetUsername() + "to " + message.GetRoom().GetName() + " Received"}, nil
		}
		return &pb.Reply{IsSuccess: false, Message: "No such room"}, nil
	}
	return &pb.Reply{IsSuccess: false, Message: "Wrong Username or Password"}, nil
}

// GetMessages implementation allows client to retrieve messages of a certain room in the server
func (s *server) GetMessages(request *pb.MessageRequest, stream pb.Chat_GetMessagesServer) error {
	if checkCred(&userList, request.GetCred()) {
		for i := request.GetStartIndex(); i < request.GetEndIndex(); i++ {
			message := chatHistory.GetMessages()[i]
			if message.GetRoom().GetName() == request.GetRoom().GetName() {
				if err := stream.Send(chatHistory.GetMessages()[i]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return nil
}

// GetMessageCount implementation allows client to check the amount of messages in the server
func (s *server) GetMessagesCount(ctx context.Context, userCred *pb.UserCred) (*pb.Reply, error) {
	if checkCred(&userList, userCred) {
		return &pb.Reply{IsSuccess: true, Message: strconv.Itoa(len(chatHistory.GetMessages()))}, nil
	}
	return &pb.Reply{IsSuccess: false, Message: "Wrong Username or Password"}, nil
}

func main() {
	// Retrieve options from command line flag
	addressPtr := flag.String("a", "localhost:12000", "Address (IP address and port number of the server)")

	// Read local user list in storage
	log.Println("Loading user list...")
	userBytes := util.GetJSON("server/users.json")
	json.Unmarshal(userBytes, &userList)

	// Read local room list in storage
	log.Println("Loading room list...")
	roomBytes := util.GetJSON("server/rooms.json")
	json.Unmarshal(roomBytes, &roomList)

	// Start server
	log.Println("Starting server...")
	lis, err := net.Listen("tcp", *addressPtr)
	if err != nil {
		log.Fatalf("(ERR_LSTN) : %v", err)
	}

	// Start GRPC server
	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{})
	log.Println("Ready to serve")

	// Serve incoming client
	if err := s.Serve(lis); err != nil {
		log.Fatalf("(ERR_SRVE) : %v", err)
	}
}

// checkCred checks user submitted credentials with the one in the database
func checkCred(userList *pb.UserList, cred *pb.UserCred) bool {
	for i := 0; i < len(userList.GetUsers()); i++ {
		// User credentials is valid
		if (userList.GetUsers()[i].GetUsername() == cred.GetUsername()) && (userList.GetUsers()[i].GetPassword() == cred.GetPassword()) {
			return true
			// User credentials invalid
		} else if (userList.GetUsers()[i].GetUsername() == cred.GetUsername()) && (userList.GetUsers()[i].GetPassword() == cred.GetPassword()) && (i == len(userList.GetUsers())-1) {
			return false
		}
	}

	return false
}

// checkRoom checks user submitted room name with the one on the database
func checkRoom(roomList *pb.RoomList, room *pb.Room) bool {
	for i := 0; i < len(roomList.GetRooms()); i++ {
		// Room name is valid
		if roomList.GetRooms()[i].GetName() == room.GetName() {
			return true
			// Room name invalid
		} else if (roomList.GetRooms()[i].GetName() == room.GetName()) && (i == len(roomList.GetRooms())-1) {
			return false
		}
	}

	return false
}
