package stev_test

import (
	"os"
	"testing"
	"time"

	"github.com/rez-go/stev"
)

func TestNonPtr(t *testing.T) {
	os.Clearenv()
	var val int
	err := stev.LoadEnv("", val)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestEmpty(t *testing.T) {
	os.Clearenv()
	cfg := struct{}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
}

func TestNilTarget(t *testing.T) {
	os.Clearenv()
	var cfg *struct{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
}

func TestUnexportedField(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		myField string `env:"MY_FIELD"`
	}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
}

func TestString(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME"`
	}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "" {
		t.Errorf(`Expected "" got %q`, cfg.Name)
	}
}

func TestFieldNames(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name          string
		REST          string
		APIVersion    string
		ServerURL     string
		ModuleName    string
		MinAPIVersion string
		Area51        string
		IPV4Address   string
	}{}
	os.Setenv("NAME", "Name")
	os.Setenv("REST", "REST")
	os.Setenv("API_VERSION", "APIVersion")
	os.Setenv("SERVER_URL", "ServerURL")
	os.Setenv("MODULE_NAME", "ModuleName")
	os.Setenv("MIN_API_VERSION", "MinAPIVersion")
	os.Setenv("AREA_51", "Area51")
	os.Setenv("IPV4_ADDRESS", "IPV4Address")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Name" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.REST != "REST" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.APIVersion != "APIVersion" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.ServerURL != "ServerURL" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.ModuleName != "ModuleName" {
		t.Errorf(`Assertion failed`)
	}
	if cfg.MinAPIVersion != "MinAPIVersion" {
		t.Errorf(`Assertion failed`)
	}
	if cfg.Area51 != "Area51" {
		t.Errorf(`Assertion failed`)
	}
	if cfg.IPV4Address != "IPV4Address" {
		t.Errorf(`Assertion failed`)
	}
}

func TestStringWithValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME"`
	}{}
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Expected "Go" got %q`, cfg.Name)
	}
}

func TestStringWithPrefix(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME"`
	}{}
	os.Setenv("NAME", "Go")
	os.Setenv("PFX_NAME", "Prefixed Go")
	err := stev.LoadEnv("PFX_", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Prefixed Go" {
		t.Errorf(`Expected "Prefixed Go" got %q`, cfg.Name)
	}
}

func TestStringRequiredNoValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME,required"`
	}{}
	err := stev.LoadEnv("", &cfg)
	if err == nil {
		t.Errorf(`Expected error`)
	}
}

func TestStringRequiredWithValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME,required"`
	}{}
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Expected "Go" got %q`, cfg.Name)
	}
}

type NameOnly struct {
	Name string `env:"NAME"`
}

func TestStringWithValueNilTarget(t *testing.T) {
	os.Clearenv()
	var cfg *NameOnly
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", cfg)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestNameOnlyWithDefaultTargetPointer(t *testing.T) {
	os.Clearenv()
	cfg := &NameOnly{Name: "Not GO"}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Not GO" {
		t.Errorf(`Expected "Not GO" got %q`, cfg.Name)
	}
}

func TestNameOnlyWithDefaultTargetPointerValidValue(t *testing.T) {
	os.Clearenv()
	cfg := &NameOnly{Name: "Not GO"}
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Expected "Go" got %q`, cfg.Name)
	}
}

func TestNameOnlyWithDoubleNilTargetValid(t *testing.T) {
	os.Clearenv()
	var cfg *NameOnly
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Expected "Go" got %q`, cfg.Name)
	}
}

func TestStringDefault(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"NAME"`
	}{Name: "Not GO"}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Not GO" {
		t.Errorf(`Expected "Not GO" got %q`, cfg.Name)
	}
}

func TestStringUntaggedWithValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string
	}{}
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Expected "Go" got %q`, cfg.Name)
	}
}

func TestStringWithValueIgnore(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:"-"`
	}{}
	os.Setenv("NAME", "Go")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "" {
		t.Errorf(`Expected "" got %q`, cfg.Name)
	}
}

func TestStringAnonymous(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string `env:",squash"`
	}{}
	err := stev.LoadEnv("", &cfg)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestStringUntagged(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Name string
	}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "" {
		t.Errorf(`Expected "" got %q`, cfg.Name)
	}
}

func TestBoolUntaggedNoValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Enabled bool
	}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Enabled {
		t.Errorf(`Unexpected value`)
	}
}

func TestBoolUntaggedImplicitValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Enabled bool
	}{}
	os.Setenv("ENABLED", "")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if !cfg.Enabled {
		t.Errorf(`Unexpected value`)
	}
}

func TestBoolUntaggedTrue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Enabled bool
	}{}
	os.Setenv("ENABLED", "true")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if !cfg.Enabled {
		t.Errorf(`Unexpected value`)
	}
}

func TestBoolUntaggedFalse(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Enabled bool
	}{}
	os.Setenv("ENABLED", "false")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Enabled {
		t.Errorf(`Unexpected value`)
	}
}

type InnerStruct struct {
	Color       string
	Size        int64
	Strength    uint32
	AspectRatio float32
}

type OuterStruct struct {
	Name     string
	NamePtr  *string
	NamePtr2 *string
	NamePtr3 *string
	Inner    InnerStruct
}

type OuterStructEmbeddedInner struct {
	Name string
	InnerStruct
}

type OuterStructForcedAnonInner struct {
	Name  string
	Inner InnerStruct `env:",squash"`
}

type InnerPrefix struct {
	Color string
	Size  int64 `env:"!ABSOLUTE_SIZE"`
}

type OuterStructNoPrefixInner struct {
	Name          string
	Description   string       `env:"!ABSOLUTE_DESC"`
	WithPrefix    InnerPrefix  `env:"WITH"`
	WithoutPrefix InnerPrefix  `env:"!WITHOUT"`
	WithPtr       *InnerPrefix `env:"PTR"`
	Anon          InnerPrefix  `env:"!,squash"`
	//TODO: test squash, anon and ptr
}

func TestPrefixes(t *testing.T) {
	os.Clearenv()
	os.Setenv("NAME", "Go (no prefix)")
	os.Setenv("PFX_NAME", "Go")
	os.Setenv("ABSOLUTE_DESC", "Description")
	os.Setenv("PFX_ABSOLUTE_DESC", "Description (prefixed)")
	os.Setenv("COLOR", "BLACK")
	os.Setenv("PFX_COLOR", "WHITE")
	os.Setenv("PFX_WITH_COLOR", "RED")
	os.Setenv("WITHOUT_COLOR", "BLUE")
	os.Setenv("PFX_WITHOUT_COLOR", "GREEN")
	os.Setenv("ABSOLUTE_SIZE", "9001")
	os.Setenv("PFX_WITH_ABSOLUTE_SIZE", "9002")
	os.Setenv("PFX_PTR_COLOR", "ORANGE")
	cfg := OuterStructNoPrefixInner{}
	err := stev.LoadEnv("PFX_", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	assertStrEq(t, cfg.Name, "Go")
	assertStrEq(t, cfg.Description, "Description")
	assertStrEq(t, cfg.WithPrefix.Color, "RED")
	assertInt64Eq(t, cfg.WithPrefix.Size, 9001)
	assertStrEq(t, cfg.WithoutPrefix.Color, "BLUE")
	assertInt64Eq(t, cfg.WithoutPrefix.Size, 9001)
	assertStrEq(t, cfg.WithPtr.Color, "ORANGE")
	assertInt64Eq(t, cfg.WithPtr.Size, 9001)
}

func TestStructUntagged(t *testing.T) {
	os.Clearenv()
	cfg := OuterStruct{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.Inner.Color != "" {
		t.Errorf(`Unexpected value`)
	}
}

func TestStructUntaggedValidValues(t *testing.T) {
	os.Clearenv()
	ptr2 := "Default String 2"
	ptr3 := "Default String 3"
	cfg := OuterStruct{
		NamePtr2: &ptr2,
		NamePtr3: &ptr3,
	}
	os.Setenv("NAME", "Go")
	os.Setenv("NAME_PTR", "Pointer to String: The Second Link")
	os.Setenv("NAME_PTR_3", "Overriden String")
	os.Setenv("INNER_COLOR", "RED")
	os.Setenv("INNER_SIZE", "10")
	os.Setenv("INNER_STRENGTH", "9001")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Unexpected value`)
	}
	if *cfg.NamePtr != "Pointer to String: The Second Link" {
		t.Errorf(`Assertion failed`)
	}
	if *cfg.NamePtr2 != "Default String 2" {
		t.Errorf(`Assertion failed`)
	}
	if *cfg.NamePtr3 != "Overriden String" {
		t.Errorf(`Assertion failed`)
	}
	if cfg.Inner.Color != "RED" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.Inner.Size != 10 {
		t.Errorf(`Assertion failed`)
	}
	if cfg.Inner.Strength != 9001 {
		t.Errorf(`Assertion failed`)
	}
}

func TestEmbeddedStructUntaggedValidValues(t *testing.T) {
	os.Clearenv()
	cfg := OuterStructEmbeddedInner{}
	os.Setenv("NAME", "Go")
	os.Setenv("COLOR", "RED")
	os.Setenv("ASPECT_RATIO", "1.3333")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.Color != "RED" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.AspectRatio != 1.3333 {
		t.Errorf(`Assertion failed`)
	}
}

func TestStructForcedAnonInnerUntaggedValidValues(t *testing.T) {
	os.Clearenv()
	cfg := OuterStructForcedAnonInner{}
	os.Setenv("NAME", "Go")
	os.Setenv("COLOR", "RED")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Name != "Go" {
		t.Errorf(`Unexpected value`)
	}
	if cfg.Inner.Color != "RED" {
		t.Errorf(`Unexpected value`)
	}
}

func TestDurationUntaggedNoValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Whatever time.Duration
	}{}
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Whatever != 0 {
		t.Errorf(`Expected 0 got %v`, cfg.Whatever)
	}
}

func TestDurationUntaggedWithValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Delay time.Duration
	}{}
	os.Setenv("DELAY", "60s")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if cfg.Delay != 60*time.Second {
		t.Errorf(`Expected 60s got %v`, cfg.Delay)
	}
}

func TestDurationPtrUntaggedWithValue(t *testing.T) {
	os.Clearenv()
	cfg := struct {
		Delay *time.Duration
	}{}
	os.Setenv("DELAY", "60s")
	err := stev.LoadEnv("", &cfg)
	if err != nil {
		t.Errorf("Expected nil, got %#v", err)
	}
	if *cfg.Delay != 60*time.Second {
		t.Errorf(`Expected 60s got %v`, cfg.Delay)
	}
}

func assertStrEq(t *testing.T, have, wanted string) {
	if wanted != have {
		t.Errorf("Assertion failed:\n\twanted: %v\n\thave:   %v", wanted, have)
	}
}
func assertInt64Eq(t *testing.T, have, wanted int64) {
	if wanted != have {
		t.Errorf("Assertion failed:\n\twanted: %v\n\thave:   %v", wanted, have)
	}
}
