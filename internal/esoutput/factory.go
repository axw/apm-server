package esoutput

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/go-elasticsearch/v7"
)

func init() {
	outputs.RegisterType("go-elasticsearch", makeOutput)
}

func makeOutput(
	indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{})
	if err != nil {
		return outputs.Group{}, err
	}
	indexSelector, pipelineSelector, err := buildSelectors(indexManager, beat, cfg)
	if err != nil {
		return outputs.Group{}, err
	}
	client, err := newClient(es, indexSelector, pipelineSelector)
	if err != nil {
		return outputs.Group{}, err
	}
	clients := []outputs.NetworkClient{client}
	return outputs.SuccessNet(true, 1, 0, clients)
}

func buildSelectors(
	im outputs.IndexManager,
	beat beat.Info,
	cfg *common.Config,
) (index outputs.IndexSelector, pipeline *outil.Selector, err error) {
	index, err = im.BuildSelector(cfg)
	if err != nil {
		return index, pipeline, err
	}

	pipelineSel, err := buildPipelineSelector(cfg)
	if err != nil {
		return index, pipeline, err
	}

	if !pipelineSel.IsEmpty() {
		pipeline = &pipelineSel
	}

	return index, pipeline, err
}

func buildPipelineSelector(cfg *common.Config) (outil.Selector, error) {
	return outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "pipeline",
		MultiKey:         "pipelines",
		EnableSingleOnly: true,
		FailEmpty:        false,
		Case:             outil.SelectorLowerCase,
	})
}
