package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"context"

	"protocol-buffers/todo"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

func main() {
	var tasks taskServer

	srv := grpc.NewServer()
	todo.RegisterTasksServer(srv, tasks)

	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("could not listen to :8888: %v", err)
	}
	log.Fatal(srv.Serve(l))
}

const dbPath = "database.db"

type taskServer struct {
	todo.UnimplementedTasksServer
}

func (t taskServer) List(ctx context.Context, void *todo.Void) (*todo.TaskLists, error) {
	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", dbPath, err)
	}

	var tasks todo.TaskLists
	for {
		if len(b) == 0 {
			return &tasks, nil
		} else if len(b) < 4 {
			return nil, fmt.Errorf("remaining odd %d bytes, what to do?", len(b))
		}

		var length int64
		if err := gob.NewDecoder(bytes.NewReader(b[:4])).Decode(&length); err != nil {
			return nil, fmt.Errorf("could not decode message length: %v", err)
		}
		b = b[4:]

		var task todo.Task
		if err := proto.Unmarshal(b[:length], &task); err != nil {
			return nil, fmt.Errorf("could not read task: %v", err)
		}
		b = b[length:]
		tasks.Tasks = append(tasks.Tasks, &task)
	}
}

func (t taskServer) Add(ctx context.Context, text *todo.Text) (*todo.Task, error) {
	task := &todo.Task{
		Text: text.Text,
		Done: false,
	}

	b, err := proto.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("could not encode task: %v", err)
	}

	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %v", dbPath, err)
	}

	if err := gob.NewEncoder(f).Encode(int64(len(b))); err != nil {
		return nil, fmt.Errorf("could not encode length of message: %v", err)
	}

	_, err = f.Write(b)
	if err != nil {
		return nil, fmt.Errorf("could not write task to file: %v", err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("could not close file %s: %v", dbPath, err)
	}

	return task, nil
}
