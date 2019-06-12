package main

import (
	"context"
	"fmt"
	"github.com/therecipe/qt/widgets"
	"google.golang.org/grpc"
	"regexp"
	"time"

	pb "./proto"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"log"
	"os"
)

type MainWindow struct {
	*widgets.QWidget

	// groupBox
	addressGroup *widgets.QGroupBox
	reqGroup     *widgets.QGroupBox
	respGroup    *widgets.QGroupBox
}

func NewMainWindow(app *widgets.QApplication) (mainWindow *MainWindow) {
	// mainWindow
	mainWindow = &MainWindow{}
	mainWindow.QWidget = widgets.NewQWidget(nil, 0)
	mainWindow.SetMinimumHeight(800)
	mainWindow.SetMinimumWidth(600)
	mainWindow.SetWindowTitle("Test GRPC")

	mainWindow.addressGroup = widgets.NewQGroupBox2("address", nil)
	mainWindow.reqGroup = widgets.NewQGroupBox2("request", nil)
	mainWindow.respGroup = widgets.NewQGroupBox2("response", nil)

	// addressGroup
	addressLabel := widgets.NewQLabel2("server address:", nil, 0)
	addressLineEdit := widgets.NewQLineEdit2("", nil)
	addressLayout := widgets.NewQGridLayout2()
	addressLayout.AddWidget(addressLabel, 0, 0, 0)
	addressLayout.AddWidget(addressLineEdit, 0, 1, 0)
	mainWindow.addressGroup.SetLayout(addressLayout)

	//respGroup
	respText := widgets.NewQTextEdit2("respText", nil)
	respLayout := widgets.NewQGridLayout2()
	respLayout.AddWidget(respText, 0, 0, 0)
	mainWindow.respGroup.SetLayout(respLayout)

	// reqGroup
	listServicesButton := widgets.NewQPushButton2("listServices", nil)
	echoButton := widgets.NewQPushButton2("echo", nil)
	reqLayout := widgets.NewQGridLayout2()
	reqLayout.AddWidget(listServicesButton, 0, 0, 0)
	reqLayout.AddWidget(echoButton, 0, 1, 0)
	mainWindow.reqGroup.SetLayout(reqLayout)

	// mainWindow layout
	grid := *widgets.NewQGridLayout2()
	grid.AddWidget(mainWindow.addressGroup, 0, 0, 0)
	grid.AddWidget(mainWindow.reqGroup, 1, 0, 0)
	grid.AddWidget(mainWindow.respGroup, 2, 0, 0)

	mainWindow.SetLayout(&grid)

	// button clicked function
	listServicesButton.ConnectClicked(func(checked bool) {
		log.Println("listServices clicked")
		resp := listServices(connect(addressLineEdit.Text()))
		respText.SetText(resp)
	})

	echoButton.ConnectClicked(func(checked bool) {
		log.Println("echo clicked")
		ping(respText, addressLineEdit.Text())
	})

	return
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func connect(address string) *grpc.ClientConn {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	check(err)
	return conn
}

func client(address string) *pb.EchoClient {
	if match, _ := regexp.MatchString(":", address); match == false {
		log.Fatalln("invalid address")
	}

	conn := connect(address)

	client := pb.NewEchoClient(conn)
	return &client
}

func listServices(conn *grpc.ClientConn) string {
	rc := rpb.NewServerReflectionClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, _ := rc.ServerReflectionInfo(ctx)
	req := &rpb.ServerReflectionRequest{MessageRequest: &rpb.ServerReflectionRequest_ListServices{ListServices: "*"}}
	err := r.Send(req)
	if err == nil {
		resp, err := r.Recv()
		if err == nil {
			return fmt.Sprint(resp.MessageResponse)
		}
	}
	return fmt.Sprint(err)
}

func ping(label *widgets.QTextEdit, address string) {
	log.Println("ping started")
	defer log.Println("ping exited")

	client := *client(address)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := pb.Request{Name: "test"}

	resp, err := client.Receive(ctx, &req)
	check(err)
	log.Println(resp)
	label.SetText(fmt.Sprintf("Response: %v", resp.Msg))
}

func main() {
	app := widgets.NewQApplication(len(os.Args), os.Args)
	mainWindow := NewMainWindow(app)
	mainWindow.Show()

	code := widgets.QApplication_Exec()
	log.Printf("QApplication exited with code: %d\n", code)
	os.Exit(code)
}
