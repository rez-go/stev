package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rez-go/stev"
	"github.com/rez-go/stev/docgen"
)

func main() {
	os.Setenv("CLIENT_ID", "HELLO")
	os.Setenv("SERVER_REST_BASE_URL", "http://localhost:8080/api/v1")
	os.Setenv("TEST_DURATION", "20s")
	os.Setenv("TEST_DURATION_PTR", "10s")

	cfg := ServiceClientConfig{
		ServiceClientCredentials: ServiceClientCredentials{
			ClientSecret: "DEF",
		},
		MapOfStruct: map[string]interface{}{
			"hello": &StructConfig{},
		},
	}

	prefix := ""

	if len(os.Args) > 1 && os.Args[1] == "env_file_template" {
		genConfigTemplate(prefix, cfg)
		return
	}

	err := stev.LoadFromEnv(prefix, &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Out %#v %v\n", cfg, cfg.TestDurationPtr.String())
}

func genConfigTemplate(prefix string, skeleton ServiceClientConfig) {
	err := docgen.WriteEnvTemplate(os.Stdout, &skeleton, docgen.EnvTemplateOptions{
		FieldPrefix: prefix,
	})
	if err != nil {
		panic(err)
	}
}

type ServiceClientCredentials struct {
	ClientID     string `env:",required"`
	ClientSecret string
}

func (ServiceClientCredentials) StevFieldDescriptions() map[string]string {
	return map[string]string{
		"ClientID":     "The client-id as provided by IAM server.",
		"TestDuration": "A record type is a data type that describes such values and variables. Most modern computer languages allow the programmer to define new record types. The definition includes specifying the data type of each field and an identifier (name or label) by which it can be accessed. In type theory, product types (with no field names) are generally preferred due to their simplicity, but proper record types are studied in languages such as System F-sub. Since type-theoretical records may contain first-class function-typed fields in addition to data, they can express many features of object-oriented programming. ",
	}
}

type ServiceClientConfig struct {
	ServerRESTBaseURL string
	ServiceClientCredentials
	TestDuration    time.Duration          `env:"TEST_DURATION"`
	TestDurationPtr *time.Duration         `env:"TEST_DURATION_PTR"`
	Struct          StructConfig           `env:"STRUCT"`
	MapOfStruct     map[string]interface{} `env:",map"`
}

type StructConfig struct {
	Name string
}
