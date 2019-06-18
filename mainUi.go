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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"io"
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
	configGroup  *widgets.QGroupBox
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
	mainWindow.configGroup = widgets.NewQGroupBox2("config", nil)
	mainWindow.reqGroup = widgets.NewQGroupBox2("request", nil)
	mainWindow.respGroup = widgets.NewQGroupBox2("response", nil)

	// addressGroup
	addressLabel := widgets.NewQLabel2("server address:", nil, 0)
	addressLineEdit := widgets.NewQLineEdit2("localhost:10000", nil)
	addressLayout := widgets.NewQGridLayout2()
	addressLayout.AddWidget(addressLabel, 0, 0, 0)
	addressLayout.AddWidget(addressLineEdit, 0, 1, 0)
	mainWindow.addressGroup.SetLayout(addressLayout)

	// configGroup
	plainTextButton := widgets.NewQRadioButton2("plainText(no TLS)", nil)
	plainTextButton.SetCheckedDefault(true)
	tlsButton := widgets.NewQRadioButton2("TLS", nil)
	serverLabel := widgets.NewQLabel2("server name", nil, 0)
	publicKeyLabel := widgets.NewQLabel2("public crt file path", nil, 0)
	privateKeyLabel := widgets.NewQLabel2("private key file path", nil, 0)
	serverName := widgets.NewQLineEdit2("", nil)
	serverName.SetDisabledDefault(true)
	publicKey := widgets.NewQLineEdit2("*", nil)
	publicKey.SetDisabledDefault(true)
	privateKey := widgets.NewQLineEdit2("*", nil)
	privateKey.SetDisabledDefault(true)
	configGroupLayout := widgets.NewQGridLayout2()
	configGroupLayout.AddWidget(plainTextButton, 0, 0, 0)
	configGroupLayout.AddWidget(tlsButton, 0, 1, 0)
	configGroupLayout.AddWidget(serverLabel, 1, 1, 0)
	configGroupLayout.AddWidget(serverName, 1, 2, 0)
	configGroupLayout.AddWidget(publicKeyLabel, 2, 1, 0)
	configGroupLayout.AddWidget(publicKey, 2, 2, 0)
	configGroupLayout.AddWidget(privateKeyLabel, 3, 1, 0)
	configGroupLayout.AddWidget(privateKey, 3, 2, 0)
	mainWindow.configGroup.SetLayout(configGroupLayout)

	//respGroup
	respText := widgets.NewQTextEdit2("respText", nil)
	respListGroup := widgets.NewQGroupBox2("list", nil)
	respList := widgets.NewQListWidget(nil)
	respListOp := widgets.NewQListWidget(nil)
	respListOpOp := widgets.NewQTextEdit2("", nil)
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
	sendCheckBox := widgets.NewQCheckBox2("message", nil)
	sendCheckBox.SetCheckedDefault(false)
	sendText := widgets.NewQTextEdit2("message in json", nil)
	sendText.SetDisabledDefault(true)
	methodNameLabel := widgets.NewQLabel2("methodName", nil, 0)
	methodName := widgets.NewQLineEdit2("service.method", nil)
	methodName.SetDisabledDefault(true)
	sendButton := widgets.NewQPushButton2("send", nil)
	sendButton.SetDisabled(true)
	reqLayout := widgets.NewQGridLayout2()
	reqLayout.AddWidget(describeButton, 0, 0, 0)
	reqLayout.AddWidget(listServicesButton, 0, 1, 1)
	reqLayout.AddWidget(sendCheckBox, 1, 0, 0)
	reqLayout.AddWidget(sendText, 1, 1, 0)
	reqLayout.AddWidget(methodNameLabel, 2, 0, 0)
	reqLayout.AddWidget(methodName, 2, 1, 0)
	reqLayout.AddWidget(sendButton, 3, 1, 0)
	mainWindow.reqGroup.SetLayout(reqLayout)

	// mainWindow layout
	grid := *widgets.NewQGridLayout2()
	grid.AddWidget(mainWindow.addressGroup, 0, 0, 0)
	grid.AddWidget(mainWindow.configGroup, 1, 0, 0)
	grid.AddWidget(mainWindow.reqGroup, 2, 0, 0)
	grid.AddWidget(mainWindow.respGroup, 3, 0, 0)

	mainWindow.SetLayout(&grid)

	// button clicked function
	tlsButton.ConnectClicked(func(checked bool) {
		serverName.SetDisabled(false)
		publicKey.SetDisabled(false)
		privateKey.SetDisabled(false)
	})

	plainTextButton.ConnectClicked(func(checked bool) {
		serverName.SetDisabled(true)
		publicKey.SetDisabled(true)
		privateKey.SetDisabled(true)
	})

	sendCheckBox.ConnectClicked(func(checked bool) {
		sendText.SetDisabled(!sendCheckBox.IsChecked())
		sendButton.SetDisabled(!sendCheckBox.IsChecked())
		methodName.SetDisabled(!sendCheckBox.IsChecked())
	})

	describeButton.ConnectClicked(func(checked bool) {
		resp := describe(addressLineEdit.Text(), plainTextButton.IsChecked(), serverName.Text(), &CA{false, "", publicKey.Text(), privateKey.Text()})
		respText.SetText(resp)
	})

	listServicesButton.ConnectClicked(func(checked bool) {
		resp := listServices(addressLineEdit.Text(), plainTextButton.IsChecked(), serverName.Text(), &CA{false, "", publicKey.Text(), privateKey.Text()})
		respText.SetText(resp)
		s := strings.Split(resp, "\n")
		respList.Clear()
		for _, i := range s[:len(s)-1] {
			newListItem := widgets.NewQListWidgetItem2(i, nil, 0)
			respList.AddItem2(newListItem)
		}
	})

	respList.ConnectClicked(func(index *core.QModelIndex) {
		// log.Println(index.Row())
		svc := respList.SelectedItems()[0].Text()
		resp := listMethods(addressLineEdit.Text(), svc, plainTextButton.IsChecked(), serverName.Text(), &CA{false, "", publicKey.Text(), privateKey.Text()})
		s := strings.Split(resp, "\n")
		respListOp.Clear()
		for _, i := range s[:len(s)-1] {
			newListItem := widgets.NewQListWidgetItem2(i, nil, 0)
			respListOp.AddItem2(newListItem)
		}
	})

	respListOp.ConnectClicked(func(index *core.QModelIndex) {
		// log.Println(index.Row())
		method := respListOp.SelectedItems()[0].Text()
		resp := methodDetails(addressLineEdit.Text(), method, plainTextButton.IsChecked(), serverName.Text(), &CA{false, "", publicKey.Text(), privateKey.Text()})
		respListOpOp.SetText(resp)
	})

	sendButton.ConnectClicked(func(checked bool) {
		methodName := methodName.Text()
		if methodName != "" {
			res := invoke(addressLineEdit.Text(), plainTextButton.IsChecked(), serverName.Text(), &CA{false, publicKey.Text(), "", privateKey.Text()}, methodName, sendText.ToPlainText())
			respText.SetText(res)
		} else {
			return
		}
	})

	return
}

func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

type CA struct {
	insecure        bool
	cacert          string // file name string
	pubKey, privKey string
}

func generateCreds(plainText bool, serverName string, ca *CA) (creds credentials.TransportCredentials, err error) {
	if !plainText {
		creds, err = grpcurl.ClientTransportCredentials(ca.insecure, ca.cacert, ca.pubKey, ca.privKey)
		if err != nil {
			return creds, fmt.Errorf("Failed to configure transport credentials due to: %s\n", err.Error())
		}
		if serverName != "" {
			if err := creds.OverrideServerName(serverName); err != nil {
				return creds, fmt.Errorf("Failed to override server name as %q due to: %s\n", serverName, err.Error())
			}
		}
	}
	return
}

func dial(ctx context.Context, address string, creds credentials.TransportCredentials) (*grpc.ClientConn, context.Context, error) {
	cc, err := grpcurl.BlockingDial(ctx, "tcp", address, creds)
	if err != nil {
		err = fmt.Errorf("Failed to dial target.host %q\n%s\n", address, err.Error())
	}
	return cc, ctx, err
}

func client(ctx context.Context, address string, plainText bool, serverName string, ca *CA) (*grpcreflect.Client, context.Context, error) {
	creds, err := generateCreds(plainText, serverName, ca)
	cc, ctx, err := dial(ctx, address, creds)
	refClient := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(cc))
	return refClient, ctx, err
}

func descSource(address string, plainText bool, serverName string, ca *CA) (grpcurl.DescriptorSource, context.CancelFunc, error) {
	dialTime := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), dialTime)
	// defer cancel()
	refClient, ctx, err := client(ctx, address, plainText, serverName, ca)
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	return descSource, cancel, err
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

func capture() func() (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	done := make(chan error, 1)

	save := os.Stdout
	os.Stdout = w

	var buf strings.Builder

	go func() {
		_, err := io.Copy(&buf, r)
		_ = r.Close()
		done <- err
	}()

	return func() (string, error) {
		os.Stdout = save
		_ = w.Close()
		err := <-done
		return buf.String(), err
	}
}

func invoke(address string, plainText bool, serverName string, ca *CA, methodName, msg string) string {
	dialTime := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), dialTime)
	defer cancel()
	creds, err := generateCreds(plainText, serverName, ca)
	if err != nil {
		return err.Error()
	}
	cc, ctx, err := dial(ctx, address, creds)
	if err != nil {
		return err.Error()
	}
	refClient := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(cc))
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	rf, formatter, err := grpcurl.RequestParserAndFormatterFor(grpcurl.Format("json"), descSource, true, true, strings.NewReader(msg))
	if err != nil {
		return fmt.Sprintf("Failed to construct request parser and formatter for json due to: %s\n", err.Error())
	}
	done := capture()
	h := grpcurl.NewDefaultEventHandler(os.Stdout, descSource, formatter, false)
	err = grpcurl.InvokeRPC(ctx, descSource, cc, methodName, []string{}, h, rf.Next)

	if err != nil {
		return fmt.Sprintf("Error invoking method %s due to: %s\n", methodName, err.Error())
	}

	if h.Status.Code() != codes.OK {
		return fmt.Sprint(h.Status)
	}

	str, err := done()
	if err != nil {
		return fmt.Sprintf("Error invoking method %s due to: %s\n", methodName, err.Error())
	}
	return str
}

func describe(address string, plainText bool, serverName string, ca *CA) string {
	ds, cancel, err := descSource(address, plainText, serverName, ca)
	if err != nil {
		return err.Error()
	}
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

func listServices(address string, plainText bool, serverName string, ca *CA) string {
	ds, cancel, err := descSource(address, plainText, serverName, ca)
	if err != nil {
		return err.Error()
	}
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

func listMethods(address, serviceName string, plainText bool, serverName string, ca *CA) string {
	ds, cancel, err := descSource(address, plainText, serverName, ca)
	if err != nil {
		return err.Error()
	}
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

func methodDetails(address, methodName string, plainText bool, serverName string, ca *CA) string {
	ds, cancel, err := descSource(address, plainText, serverName, ca)
	if err != nil {
		return err.Error()
	}
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
