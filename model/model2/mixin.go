package model

import (
	"github.com/elastic/apm-server/model/modelpb"
)

type anyValue struct {
	pb  modelpb.AnyValue
	set bool
}

type keyValue struct {
	pb modelpb.KeyValue
}

type mixinMetadata struct {
}

type mixinService struct {
}

type mixinServiceNode struct {
}

type mixinLanguage struct {
}

type mixinRuntime struct {
}

type mixinFramework struct {
}

type mixinAgent struct {
}

type mixinProcess struct {
}

type mixinSystem struct {
}

type mixinContainer struct {
}

type mixinKubernetes struct {
}

type mixinKubernetesNode struct {
}

type mixinKubernetesPod struct {
}

type mixinUser struct {
}

type mixinUserAgent struct {
}
