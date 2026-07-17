package admin

import (
	"bytes"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminTemplateRenders(t *testing.T) {
	var output bytes.Buffer
	if err := adminTemplate.Execute(&output, gin.H{}); err != nil {
		t.Fatalf("render admin template: %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("admin template rendered an empty response")
	}
}
