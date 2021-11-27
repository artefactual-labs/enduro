package amclient

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DemoRepository struct {
	packages []*PackageStub
	mu       sync.RWMutex
}

type PackageStub struct {
	Name       string
	Type       string
	TransferID string
	SIPID      string
	CreatedAt  time.Time
}

var dc = &DemoRepository{
	packages: []*PackageStub{},
}

func DemoizeClient(c *Client) {
	c.Transfer = &DemoTransferService{client: dc}
	c.Ingest = &DemoIngestService{client: dc}
	c.ProcessingConfig = &DemoProcessingConfigService{client: dc}
	c.Package = &DemoPackageService{client: dc}
	c.Jobs = &DemoJobsService{client: dc}
	c.Task = &DemoTasksService{client: dc}
}

func StubResponseStatusOK() (*Response, error) {
	resp := &http.Response{
		Proto:      "http",
		Status:     "OK",
		StatusCode: http.StatusOK,
		Header:     http.Header(map[string][]string{}),
	}
	return &Response{resp}, CheckResponse(resp)
}

func StubResponseStatusNotFound() (*Response, error) {
	resp := &http.Response{
		Proto:      "http",
		Status:     "Error",
		StatusCode: http.StatusNotFound,
		Header:     map[string][]string{},
	}
	return &Response{resp}, CheckResponse(resp)
}

func StubResponseStatusNotImplemented() (*Response, error) {
	resp := &http.Response{
		Proto:      "http",
		Status:     "Not implemented",
		StatusCode: http.StatusNotImplemented,
		Header:     map[string][]string{},
	}
	return &Response{resp}, CheckResponse(resp)
}

type DemoTransferService struct {
	client *DemoRepository
}

func (svc *DemoTransferService) Start(context.Context, *TransferStartRequest) (*TransferStartResponse, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return &TransferStartResponse{}, resp, err
}

func (svc *DemoTransferService) Approve(context.Context, *TransferApproveRequest) (*TransferApproveResponse, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return &TransferApproveResponse{}, resp, err
}

func (svc *DemoTransferService) Unapproved(context.Context, *TransferUnapprovedRequest) (*TransferUnapprovedResponse, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return &TransferUnapprovedResponse{}, resp, err
}

func (svc *DemoTransferService) Status(ctx context.Context, ID string) (*TransferStatusResponse, *Response, error) {
	svc.client.mu.Lock()
	defer svc.client.mu.Unlock()

	var match *PackageStub
	for _, pkg := range svc.client.packages {
		if pkg.TransferID == ID {
			match = pkg
			break
		}
	}

	if match == nil {
		resp, err := StubResponseStatusNotFound()
		return &TransferStatusResponse{}, resp, err
	}

	match.SIPID = uuid.New().String()

	payload := &TransferStatusResponse{
		ID:     match.TransferID,
		SIPID:  match.SIPID,
		Status: "COMPLETE",
	}

	resp, err := StubResponseStatusOK()
	return payload, resp, err
}

func (svc *DemoTransferService) Hide(ctx context.Context, ID string) (*TransferHideResponse, *Response, error) {
	resp, err := StubResponseStatusOK()
	return &TransferHideResponse{Removed: true}, resp, err
}

type DemoIngestService struct {
	client *DemoRepository
}

func (svc *DemoIngestService) Status(ctx context.Context, ID string) (*IngestStatusResponse, *Response, error) {
	svc.client.mu.Lock()
	defer svc.client.mu.Unlock()

	var match *PackageStub
	for _, pkg := range svc.client.packages {
		if pkg.SIPID == ID {
			match = pkg
			break
		}
	}

	if match == nil {
		resp, err := StubResponseStatusNotFound()
		return &IngestStatusResponse{}, resp, err
	}

	payload := &IngestStatusResponse{
		ID:     ID,
		Status: "COMPLETE",
	}

	resp, err := StubResponseStatusOK()
	return payload, resp, err
}

func (svc *DemoIngestService) Hide(ctx context.Context, ID string) (*IngestHideResponse, *Response, error) {
	resp, err := StubResponseStatusOK()
	return &IngestHideResponse{Removed: true}, resp, err
}

type DemoProcessingConfigService struct {
	client *DemoRepository
}

func (svc *DemoProcessingConfigService) Get(context.Context, string) (*ProcessingConfig, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return &ProcessingConfig{}, resp, err
}

type DemoPackageService struct {
	client *DemoRepository
}

func (svc *DemoPackageService) Create(ctx context.Context, req *PackageCreateRequest) (*PackageCreateResponse, *Response, error) {
	svc.client.mu.Lock()
	defer svc.client.mu.Unlock()

	pkg := PackageStub{
		Name:       req.Name,
		Type:       req.Type,
		TransferID: uuid.New().String(),
		CreatedAt:  time.Now().UTC(),
	}
	svc.client.packages = append(svc.client.packages, &pkg)

	payload := &PackageCreateResponse{ID: pkg.TransferID}

	resp, err := StubResponseStatusOK()
	return payload, resp, err
}

type DemoJobsService struct {
	client *DemoRepository
}

func (svc *DemoJobsService) List(context.Context, string, *JobsListRequest) ([]Job, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return []Job{}, resp, err
}

type DemoTasksService struct {
	client *DemoRepository
}

func (svc *DemoTasksService) Read(context.Context, string) (*TaskDetailed, *Response, error) {
	resp, err := StubResponseStatusNotImplemented()
	return &TaskDetailed{}, resp, err
}
