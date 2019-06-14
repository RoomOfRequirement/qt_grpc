package main

import (
	"context"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"regexp"
	"strings"
	"time"
	"unsafe"

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
	mainWindow.SetWindowTitle("GRPC Descriptor")

	mainWindow.addressGroup = widgets.NewQGroupBox2("address", nil)
	mainWindow.reqGroup = widgets.NewQGroupBox2("request", nil)
	mainWindow.respGroup = widgets.NewQGroupBox2("response", nil)

	// addressGroup
	addressLabel := widgets.NewQLabel2("server address:", nil, 0)
	addressLineEdit := widgets.NewQLineEdit2("localhost:10000", nil)
	addressLayout := widgets.NewQGridLayout2()
	addressLayout.AddWidget(addressLabel, 0, 0, 0)
	addressLayout.AddWidget(addressLineEdit, 0, 1, 0)
	mainWindow.addressGroup.SetLayout(addressLayout)

	//respGroup
	respText := widgets.NewQTextEdit2("respText", nil)
	respListGroup := widgets.NewQGroupBox2("list", nil)
	respList := widgets.NewQListWidget(nil)
	respListOp := widgets.NewQListWidget(nil)
	respListOpOp := widgets.NewQListWidget(nil)
	respListGroupLayout := widgets.NewQGridLayout2()
	respListGroupLayout.AddWidget(respList, 0, 1, 0)
	respListGroupLayout.AddWidget(respListOp, 1, 1, 0)
	respListGroupLayout.AddWidget(respListOpOp, 2, 1, 0)
	respListGroup.SetLayout(respListGroupLayout)

	respLayout := widgets.NewQGridLayout2()
	respLayout.AddWidget(respText, 0, 0, 0)
	respLayout.AddWidget(respListGroup, 0, 1, 0)
	mainWindow.respGroup.SetLayout(respLayout)

	// reqGroup
	describeButton := widgets.NewQPushButton2("describeServer", nil)
	listServicesButton := widgets.NewQPushButton2("listServices", nil)
	reqLayout := widgets.NewQGridLayout2()
	reqLayout.AddWidget(describeButton, 0, 0, 0)
	reqLayout.AddWidget(listServicesButton, 1, 0, 0)
	mainWindow.reqGroup.SetLayout(reqLayout)

	// mainWindow layout
	grid := *widgets.NewQGridLayout2()
	grid.AddWidget(mainWindow.addressGroup, 0, 0, 0)
	grid.AddWidget(mainWindow.reqGroup, 1, 0, 0)
	grid.AddWidget(mainWindow.respGroup, 2, 0, 0)

	mainWindow.SetLayout(&grid)

	// button clicked function
	describeButton.ConnectClicked(func(checked bool) {
		resp := describe(addressLineEdit.Text())
		respText.SetText(resp)
	})

	listServicesButton.ConnectClicked(func(checked bool) {
		resp := listServices(addressLineEdit.Text())
		respText.SetText(resp)
		s := strings.Split(resp, "\n")
		respList.Clear()
		for _, i := range s {
			newListItem := widgets.NewQListWidgetItem2(i, nil, 0)
			respList.AddItem2(newListItem)
		}
	})

	respList.ConnectClicked(func(index *core.QModelIndex) {
		// log.Println(index.Row())
		svc := respList.SelectedItems()[0].Text()
		resp := listMethods(addressLineEdit.Text(), svc)
		s := strings.Split(resp, "\n")
		respListOp.Clear()
		for _, i := range s {
			newListItem := widgets.NewQListWidgetItem2(i, nil, 0)
			respListOp.AddItem2(newListItem)
		}
	})

	respListOp.ConnectClicked(func(index *core.QModelIndex) {
		// log.Println(index.Row())
		method := respListOp.SelectedItems()[0].Text()
		resp := methodDetails(addressLineEdit.Text(), method)
		s := strings.Split(resp, "\n")
		respListOpOp.Clear()
		for _, i := range s {
			newListItem := widgets.NewQListWidgetItem2(i, nil, 0)
			respListOpOp.AddItem2(newListItem)
		}
	})

	return
}

func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func dial(ctx context.Context, address string) (*grpc.ClientConn, context.Context) {
	var creds credentials.TransportCredentials
	cc, err := grpcurl.BlockingDial(ctx, "tcp", address, creds)
	if err != nil {
		log.Printf("Failed to dial target.host %q\n", address)
		log.Fatalln(err.Error())
	}
	return cc, ctx
}

func client(ctx context.Context, address string) (*grpcreflect.Client, context.Context) {
	cc, ctx := dial(ctx, address)
	refClient := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(cc))
	return refClient, ctx
}

func descSource(address string) (grpcurl.DescriptorSource, context.CancelFunc) {
	dialTime := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), dialTime)
	// defer cancel()
	refClient, ctx := client(ctx, address)
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	return descSource, cancel
}

func parseReq(symbols []string, ds grpcurl.DescriptorSource) string {
	res := ""
	for _, s := range symbols {
		if s[0] == '.' {
			s = s[1:]
		}

		dsc, err := ds.FindSymbol(s)
		if err != nil {
			return fmt.Sprintf("Failed to resolve symbol %q due to %s\n", s, err.Error())
		}

		fqn := dsc.GetFullyQualifiedName()
		var elementType string
		switch d := dsc.(type) {
		case *desc.MessageDescriptor:
			elementType = "a message"
			parent, ok := d.GetParent().(*desc.MessageDescriptor)
			if ok {
				if d.IsMapEntry() {
					for _, f := range parent.GetFields() {
						if f.IsMap() && f.GetMessageType() == d {
							// found it: describe the map field instead
							elementType = "the entry type for a map field"
							dsc = f
							break
						}
					}
				} else {
					// see if it's a group
					for _, f := range parent.GetFields() {
						if f.GetType() == descpb.FieldDescriptorProto_TYPE_GROUP && f.GetMessageType() == d {
							// found it: describe the map field instead
							elementType = "the type of a group field"
							dsc = f
							break
						}
					}
				}
			}
		case *desc.FieldDescriptor:
			elementType = "a field"
			if d.GetType() == descpb.FieldDescriptorProto_TYPE_GROUP {
				elementType = "a group field"
			} else if d.IsExtension() {
				elementType = "an extension"
			}
		case *desc.OneOfDescriptor:
			elementType = "a one-of"
		case *desc.EnumDescriptor:
			elementType = "an enum"
		case *desc.EnumValueDescriptor:
			elementType = "an enum value"
		case *desc.ServiceDescriptor:
			elementType = "a service"
		case *desc.MethodDescriptor:
			elementType = "a method"
		default:
			err = fmt.Errorf("descriptor has unrecognized type %T", dsc)
			return fmt.Sprintf("Failed to describe symbol %q due to %s\n", s, err.Error())
		}

		txt, err := grpcurl.GetDescriptorText(dsc, ds)
		if err != nil {
			return fmt.Sprintf("Failed to describe symbol %q due to %s\n", s, err.Error())
		}

		res += fmt.Sprintf("%s is %s:\n", fqn, elementType) + fmt.Sprintln(txt) + "\n"
	}
	return res
}

func describe(address string) string {
	ds, cancel := descSource(address)
	defer cancel()
	svcs, err := grpcurl.ListServices(ds)
	if err != nil {
		return fmt.Sprintf("Failed to list services due to:\n %s\n", err.Error())
	}
	if len(svcs) == 0 {
		return fmt.Sprint("Server returned an empty list of exposed services\n")
	}
	symbols := svcs
	res := parseReq(symbols, ds)
	return res
}

func listServices(address string) string {
	ds, cancel := descSource(address)
	defer cancel()
	svcs, err := grpcurl.ListServices(ds)
	if err != nil {
		return fmt.Sprintf("Failed to list services due to:\n %s\n", err.Error())
	}
	if len(svcs) == 0 {
		return fmt.Sprint("No services\n")
	} else {
		res := ""
		for _, svc := range svcs {
			res += svc + "\n"
		}
		return res
	}
}

func listMethods(address, serviceName string) string {
	ds, cancel := descSource(address)
	defer cancel()
	methods, err := grpcurl.ListMethods(ds, serviceName)
	if err != nil {
		return fmt.Sprintf("Failed to list methods due to:\n %s\n", err.Error())
	}
	if len(methods) == 0 {
		return fmt.Sprint("No methods\n") // probably unlikely
	} else {
		res := ""
		for _, method := range methods {
			res += method + "\n"
		}
		return res
	}
}

func methodDetails(address, methodName string) string {
	ds, cancel := descSource(address)
	defer cancel()
	symbols := []string{methodName}
	res := parseReq(symbols, ds) + "\n"
	reg := regexp.MustCompile(`\( \.(.*?) \)`)
	msgs := reg.FindAllStringSubmatch(res, -1)
	// log.Println(msgs)
	if len(msgs) != 0 {
		var tmp []string
		for _, msg := range msgs {
			// log.Println(msg[1])
			tmp = append(tmp, msg[1])
		}
		res += parseReq(tmp, ds)
	}
	return res
}

func main() {
	app := widgets.NewQApplication(len(os.Args), os.Args)
	mainWindow := NewMainWindow(app)
	mainWindow.Show()

	code := widgets.QApplication_Exec()
	log.Printf("QApplication exited with code: %d\n", code)
	os.Exit(code)
}
