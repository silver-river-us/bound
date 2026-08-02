package parser_test

import (
	. "bound/src/parser"
	"strings"
	"testing"
)

func TestBehaviorSignatureIsParsedAndValidated(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    interface Reports do
      value Snapshot do
        state :created_at :timestamp
      end
      behavior render(snapshot Snapshot, warnings string[]) returns string
    end
    exposes Reports
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	operation := architecture.Contexts["Reporting"].Interfaces["Reports"].Operations["render"]
	if len(operation.Parameters) != 2 || operation.Parameters[0].Type != "Snapshot" {
		t.Fatalf("parameters = %#v", operation.Parameters)
	}
	if operation.Returns != "string" {
		t.Fatalf("returns = %q, want string", operation.Returns)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestEntityCanDeclareBehavior(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  entity Organization do
    state :login :string
    behavior rename(login string) returns Organization
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	organization := architecture.Objects["Organization"]
	if organization == nil {
		t.Fatal("Organization was not parsed")
	}
	operation := organization.Operations["rename"]
	if len(operation.Parameters) != 1 || operation.Parameters[0].Type != "string" {
		t.Fatalf("parameters = %#v", operation.Parameters)
	}
	if operation.Returns != "Organization" {
		t.Fatalf("returns = %q, want Organization", operation.Returns)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestValueCannotDeclareBehavior(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  value Email do
    state :address :string
    behavior normalize() returns Email
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "values cannot declare behavior") {
		t.Fatalf("error = %v, want value behavior rejection", err)
	}
}

func TestBehaviorRejectsMalformedParameters(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    interface Reports do
      behavior render(snapshot) returns string
    end
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "invalid behavior parameter") {
		t.Fatalf("error = %v, want invalid behavior parameter", err)
	}
}

func TestValidationRejectsUnknownContractType(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    interface Reports do
      behavior render(snapshot Snapshop) returns string
    end
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "unknown type Snapshop") {
		t.Fatalf("error = %v, want unknown type Snapshop", err)
	}
}

func TestValidationRequiresRelationshipForCrossContextType(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Source do
    interface Public do
      value Item do
        state :id :string
      end
    end
    exposes Public
  end
  context Consumer do
    interface Reader do
      behavior read() returns Source.Public.Item
    end
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "unknown return type") {
		t.Fatalf("error = %v, want cross-context type relationship rejection", err)
	}
}
