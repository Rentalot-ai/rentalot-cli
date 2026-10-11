package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

func image33mFixtures(t *testing.T) map[string][]byte {
	t.Helper()
	photo := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range photo.Pix {
		photo.Pix[i] = byte(i)
	}
	var jpg, pngBytes bytes.Buffer
	if err := jpeg.Encode(&jpg, photo, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngBytes, photo); err != nil {
		t.Fatal(err)
	}
	// A complete 1x1 lossless WebP, rather than an extension or MIME-only stub.
	webp, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{"image/jpeg": jpg.Bytes(), "image/png": pngBytes.Bytes(), "image/webp": webp}
}

func image33mBatchFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "images.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImage33mSingleRequests(t *testing.T) {
	for contentType, photo := range image33mFixtures(t) {
		for _, operation := range []string{"presign", "confirm"} {
			t.Run(operation+"/"+contentType, func(t *testing.T) {
				metadata := map[string]any{"contentType": contentType, "sizeBytes": float64(len(photo))}
				args := []string{"p1", "--content-type", contentType, "--size-bytes", strconv.Itoa(len(photo))}
				response := `{"data":{"uploadUrl":"https://storage.example/upload","r2Key":"properties/p1/photo"}}`
				if operation == "confirm" {
					metadata["r2Key"] = "properties/p1/photo"
					metadata["altText"] = "Living room"
					args = append(args, "--r2-key", "properties/p1/photo", "--alt-text", "Living room")
					response = `{"data":{"id":"image-1","url":"https://cdn.example/photo","altText":"Living room","order":0}}`
				} else {
					metadata["fileName"] = "photo"
					args = append(args, "--file-name", "photo")
				}
				calls := 0
				out, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != http.MethodPost || r.URL.Path != "/api/v1/properties/p1/images/"+operation {
						t.Errorf("request = %s %s", r.Method, r.URL.Path)
					}
					if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Content-Type") != "application/json" {
						t.Error("missing authenticated JSON headers")
					}
					var got map[string]any
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(got, metadata) {
						t.Errorf("body = %#v, want %#v", got, metadata)
					}
					_, _ = w.Write([]byte(response))
				}, propertyImageRequestCmd(operation), args...)
				if err != nil || calls != 1 {
					t.Fatalf("calls = %d, error = %v", calls, err)
				}
				var got, want any
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(response), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("output = %s, want %s", out, response)
				}
			})
		}
	}
}

func image33mJSON(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestImage33mBatches(t *testing.T) {
	for _, operation := range []string{"presign-batch", "confirm-batch"} {
		t.Run(operation, func(t *testing.T) {
			var images []map[string]any
			for contentType, photo := range image33mFixtures(t) {
				metadata := map[string]any{"contentType": contentType, "sizeBytes": float64(len(photo))}
				if operation == "confirm-batch" {
					metadata["r2Key"] = "properties/p1/" + strings.TrimPrefix(contentType, "image/")
				} else {
					metadata["fileName"] = "photo." + strings.TrimPrefix(contentType, "image/")
				}
				images = append(images, metadata)
			}
			body := image33mJSON(t, map[string]any{"images": images})
			calls := 0
			_, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/properties/p1/images/"+operation {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				var got, want any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal(body, &want); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("body = %#v, want %#v", got, want)
				}
				_, _ = w.Write([]byte(`{"data":[]}`))
			}, propertyImageRequestCmd(operation), "p1", "--file", image33mBatchFile(t, string(body)))
			if err != nil || calls != 1 {
				t.Fatalf("calls = %d, error = %v", calls, err)
			}
		})
	}
}

func TestImage33mRejectUnsupportedBeforeHTTP(t *testing.T) {
	for _, operation := range []string{"presign", "confirm", "presign-batch", "confirm-batch"} {
		for _, contentType := range []string{"image/heic", "image/heif", "image/gif", "application/octet-stream", "image/jpeg; charset=utf-8"} {
			t.Run(operation+"/"+contentType, func(t *testing.T) {
				args := []string{"p1", "--content-type", contentType, "--size-bytes", "123"}
				switch {
				case strings.HasSuffix(operation, "-batch"):
					field := `"fileName":"photo.heic"`
					if strings.HasPrefix(operation, "confirm") {
						field = `"r2Key":"properties/p1/photo"`
					}
					args = []string{"p1", "--file", image33mBatchFile(t, `{"images":[{`+field+`,"contentType":"`+contentType+`","sizeBytes":123}]}`)}
				case operation == "confirm":
					args = append(args, "--r2-key", "properties/p1/photo")
				default:
					args = append(args, "--file-name", "photo.heic")
				}
				_, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
					t.Error("invalid content type reached API")
				}, propertyImageRequestCmd(operation), args...)
				if err == nil || !strings.Contains(err.Error(), "convert HEIC") {
					t.Fatalf("error = %v, want supported formats and conversion guidance", err)
				}
			})
		}
	}
}

func TestImage33mConfirmRejection(t *testing.T) {
	for _, operation := range []string{"confirm", "confirm-batch"} {
		for _, rejection := range []struct{ name, body, reason string }{
			{"nsfw", `{"type":"https://rentalot.ai/problems/validation-error","title":"Unprocessable Entity","status":422,"detail":"Image rejected: NSFW content detected"}`, "NSFW content detected"},
			{"invalid bytes", `{"error":{"code":"validation_error","message":"Uploaded bytes are not a valid JPEG, PNG, or WebP image"}}`, "not a valid JPEG"},
			{"unreadable response", `not JSON`, "HTTP 422"},
		} {
			t.Run(operation+"/"+rejection.name, func(t *testing.T) {
				args := []string{"p1", "--r2-key", "properties/p1/photo", "--content-type", "image/jpeg", "--size-bytes", "256"}
				if operation == "confirm-batch" {
					args = []string{"p1", "--file", image33mBatchFile(t, `{"images":[{"r2Key":"properties/p1/photo","contentType":"image/jpeg","sizeBytes":256}]}`)}
				}
				calls := 0
				out, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.WriteHeader(http.StatusUnprocessableEntity)
					_, _ = w.Write([]byte(rejection.body))
				}, propertyImageRequestCmd(operation), args...)
				if err == nil {
					t.Fatal("rejected image reported success")
				}
				for _, text := range []string{"HTTP 422", rejection.reason, "rejected upload was deleted", "presign/upload it again"} {
					if !strings.Contains(err.Error(), text) {
						t.Errorf("error = %v, missing %q", err, text)
					}
				}
				var apiErr *rentalotcli.APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 || apiErr.Retryable() {
					t.Errorf("wrapped API error = %#v", apiErr)
				}
				if calls != 1 || strings.Contains(out, `"data"`) {
					t.Errorf("calls = %d, output = %q", calls, out)
				}
			})
		}
	}
}

func TestImage33mOtherErrorsDoNotClaimDeletion(t *testing.T) {
	for _, tc := range []struct {
		operation string
		status    int
	}{{"presign", 422}, {"confirm", 400}, {"confirm", 403}, {"confirm", 500}} {
		t.Run(tc.operation+"/"+http.StatusText(tc.status), func(t *testing.T) {
			args := []string{"p1", "--content-type", "image/png", "--size-bytes", "256"}
			if tc.operation == "presign" {
				args = append(args, "--file-name", "photo.png")
			} else {
				args = append(args, "--r2-key", "properties/p1/photo")
			}
			_, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"type":"error","detail":"Request rejected"}`))
			}, propertyImageRequestCmd(tc.operation), args...)
			if err == nil || !strings.Contains(err.Error(), "Request rejected") || strings.Contains(err.Error(), "deleted") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestImage33mInvalidBatches(t *testing.T) {
	for _, body := range []string{
		`{"images":[]}`, `{"images":null}`, `{}`, `[]`, `{"images":[`,
		`{"images":[{"fileName":"photo.png","contentType":"image/png","sizeBytes":0}]}`,
		`{"images":[{"fileName":"photo.png","contentType":"image/png","sizeBytes":10485761}]}`,
		`{"images":[{"fileName":" ","contentType":"image/png","sizeBytes":256}]}`,
		`{"images":[{"fileName":"photo.png","contentType":"image/png","sizeBytes":256,"unexpected":true}]}`,
		`{"images":[{"fileName":"photo.png","contentType":"image/png","sizeBytes":256}]} {}`,
	} {
		_, err := executeCommand(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("invalid batch reached API")
		}, propertyImageRequestCmd("presign-batch"), "p1", "--file", image33mBatchFile(t, body))
		if err == nil {
			t.Fatalf("batch %s accepted", body)
		}
	}
}

func TestImage33mHelpAndRegistration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"properties", "images"})
	if err != nil || cmd.Name() != "images" {
		t.Fatalf("image commands not registered: %v", err)
	}
	for _, operation := range []string{"presign", "confirm", "presign-batch", "confirm-batch"} {
		child, _, err := cmd.Find([]string{operation})
		if err != nil || child.Name() != operation {
			t.Fatalf("%s not registered: %v", operation, err)
		}
		for _, text := range []string{"image/jpeg", "image/png", "image/webp", "HEIC is unsupported", "NSFW", "HTTP 422", "deletes"} {
			if !strings.Contains(child.Long, text) {
				t.Errorf("%s help missing %q", operation, text)
			}
		}
	}
}

func TestImage33mMetadataValidation(t *testing.T) {
	for _, confirm := range []bool{false, true} {
		valid := imageUploadRequest{FileName: "photo.jpg", ContentType: "image/jpeg", SizeBytes: 10 * 1024 * 1024}
		if confirm {
			valid.FileName = ""
			valid.R2Key = "properties/p1/photo"
		}
		if err := valid.validate(confirm); err != nil {
			t.Fatalf("maximum size rejected: %v", err)
		}
		for _, size := range []int64{-1, 0, 10*1024*1024 + 1} {
			invalid := valid
			invalid.SizeBytes = size
			if err := invalid.validate(confirm); err == nil {
				t.Errorf("size %d accepted", size)
			}
		}
		invalid := valid
		invalid.FileName, invalid.R2Key = " ", " "
		if err := invalid.validate(confirm); err == nil {
			t.Error("blank identifier accepted")
		}
		invalid = valid
		if confirm {
			invalid.FileName = "photo.jpg"
		} else {
			invalid.R2Key = "properties/p1/photo"
		}
		if err := invalid.validate(confirm); err == nil {
			t.Error("fields for wrong operation accepted")
		}
	}
}

func TestImage33mBatchLimitsAndFileErrors(t *testing.T) {
	images := make([]imageUploadRequest, 20)
	for i := range images {
		images[i] = imageUploadRequest{FileName: "photo.jpg", ContentType: "image/jpeg", SizeBytes: 256}
	}
	path := image33mBatchFile(t, string(image33mJSON(t, map[string]any{"images": images})))
	if _, err := loadImageUploadBatch(path, false); err != nil {
		t.Fatalf("20-image presign rejected: %v", err)
	}
	images = append(images, images[0])
	path = image33mBatchFile(t, string(image33mJSON(t, map[string]any{"images": images})))
	if _, err := loadImageUploadBatch(path, false); err == nil {
		t.Fatal("21-image presign accepted")
	}
	if _, err := loadImageUploadBatch(filepath.Join(t.TempDir(), "missing.json"), false); err == nil {
		t.Fatal("missing batch file accepted")
	}
	path = image33mBatchFile(t, `{"images":[{"r2Key":" ","contentType":"image/jpeg","sizeBytes":256}]}`)
	if _, err := loadImageUploadBatch(path, true); err == nil {
		t.Fatal("confirm batch with blank key accepted")
	}
}
