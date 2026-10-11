package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const imageUploadHelp = `Image uploads accept only image/jpeg, image/png, and image/webp (max 10 MB).
HEIC is unsupported; convert it to JPEG, PNG, or WebP before uploading.
Presign returns upload metadata; upload the bytes to the returned URL before confirming.
Confirm and confirm-batch screen the actual bytes for a valid JPEG, PNG, or WebP
and NSFW content. A rejection returns HTTP 422 and deletes the rejected upload.
Use a suitable replacement image and presign/upload it again; do not retry the deleted key.`

func propertiesImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Presign and confirm property image uploads",
		Long:  imageUploadHelp,
	}
	for _, operation := range []string{"presign", "confirm", "presign-batch", "confirm-batch"} {
		cmd.AddCommand(propertyImageRequestCmd(operation))
	}
	return cmd
}

// imageUploadRequest holds metadata only. The API owns byte and NSFW screening.
type imageUploadRequest struct {
	FileName    string `json:"fileName,omitempty"`
	R2Key       string `json:"r2Key,omitempty"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	AltText     string `json:"altText,omitempty"`
}

func propertyImageRequestCmd(operation string) *cobra.Command {
	confirm := strings.HasPrefix(operation, "confirm")
	batch := strings.HasSuffix(operation, "-batch")
	var image imageUploadRequest
	var file string
	cmd := &cobra.Command{
		Use:   operation + " <property-id>",
		Short: "Send an image " + operation + " request",
		Long:  imageUploadHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			if batch {
				images, err := loadImageUploadBatch(file, confirm)
				if err != nil {
					return err
				}
				body = struct {
					Images []imageUploadRequest `json:"images"`
				}{images}
			} else {
				if err := image.validate(confirm); err != nil {
					return err
				}
				body = image
			}
			path := "/api/v1/properties/" + url.PathEscape(args[0]) + "/images/" + operation
			resp, err := clientFromContext(cmd).Post(cmd.Context(), path, body)
			if err != nil {
				return fmt.Errorf("image %s: %w", operation, err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				apiErr := decodeAPIError(resp)
				if confirm && resp.StatusCode == http.StatusUnprocessableEntity {
					return fmt.Errorf("image %s: HTTP 422: %w (the rejected upload was deleted; use a suitable replacement image and presign/upload it again)", operation, apiErr)
				}
				return fmt.Errorf("image %s: %w", operation, apiErr)
			}
			return printRawJSON(resp.Body)
		},
	}
	if batch {
		cmd.Flags().StringVar(&file, "file", "", `JSON file containing {"images": [...]} (required)`)
		_ = cmd.MarkFlagRequired("file")
	} else {
		cmd.Flags().StringVar(&image.ContentType, "content-type", "", "image/jpeg, image/png, or image/webp (required; HEIC unsupported)")
		cmd.Flags().Int64Var(&image.SizeBytes, "size-bytes", 0, "image size in bytes, up to 10 MB (required)")
		_ = cmd.MarkFlagRequired("content-type")
		_ = cmd.MarkFlagRequired("size-bytes")
		if confirm {
			cmd.Flags().StringVar(&image.R2Key, "r2-key", "", "uploaded object key returned by presign (required)")
			cmd.Flags().StringVar(&image.AltText, "alt-text", "", "image alt text")
			_ = cmd.MarkFlagRequired("r2-key")
		} else {
			cmd.Flags().StringVar(&image.FileName, "file-name", "", "image file name (required)")
			_ = cmd.MarkFlagRequired("file-name")
		}
	}
	return cmd
}

func (image imageUploadRequest) validate(confirm bool) error {
	switch image.ContentType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return fmt.Errorf("unsupported image content type %q: use image/jpeg, image/png, or image/webp; convert HEIC before uploading", image.ContentType)
	}
	if image.SizeBytes <= 0 || image.SizeBytes > 10*1024*1024 {
		return fmt.Errorf("sizeBytes must be positive and at most 10 MB")
	}
	if confirm {
		if strings.TrimSpace(image.R2Key) == "" {
			return fmt.Errorf("r2Key is required for confirm")
		}
		if image.FileName != "" {
			return fmt.Errorf("fileName is only accepted by presign")
		}
	} else {
		if strings.TrimSpace(image.FileName) == "" {
			return fmt.Errorf("fileName is required for presign")
		}
		if image.R2Key != "" || image.AltText != "" {
			return fmt.Errorf("r2Key and altText are only accepted by confirm")
		}
	}
	return nil
}

func loadImageUploadBatch(path string, confirm bool) ([]imageUploadRequest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loading image batch: %w", err)
	}
	defer func() { _ = f.Close() }()
	var body struct {
		Images []imageUploadRequest `json:"images"`
	}
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf("parsing image batch: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("image batch must contain exactly one JSON object")
	}
	if len(body.Images) == 0 || (!confirm && len(body.Images) > 20) {
		return nil, fmt.Errorf("image batch must be nonempty; presign accepts at most 20 images")
	}
	for i, image := range body.Images {
		if err := image.validate(confirm); err != nil {
			return nil, fmt.Errorf("image %d: %w", i+1, err)
		}
	}
	return body.Images, nil
}
