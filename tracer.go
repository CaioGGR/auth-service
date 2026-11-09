// ==============================================================================
// OPENTELEMETRY TRACER - CONFIGURAÇÃO DE RASTREAMENTO DISTRIBUÍDO
// ==============================================================================
// Este arquivo inicializa o SDK do OpenTelemetry em Go, conectando o microsserviço
// ao OTel Collector via HTTP/OTLP (porta 4318) e configurando a propagação de
// contexto (W3C TraceContext) entre os serviços.

package main

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Lê o endereço do OTel Collector definido no ConfigMap (01-config.yaml)
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector.monitoring.svc.cluster.local:4318"
	}
	// O exporter otlptracehttp espera apenas host:porta no WithEndpoint,
	// então removemos o prefixo http:// ou https:// caso esteja presente
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Cria o exportador que enviará os Spans (traces) via HTTP para o OTel Collector
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Comunicação interna no cluster sem TLS
	)
	if err != nil {
		return nil, err
	}

	// Identifica o nome do microsserviço nos metadados do OpenTelemetry
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// Configura o TracerProvider enviando os dados em lotes (WithBatcher) para alta performance
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Registra o TracerProvider globalmente na aplicação
	otel.SetTracerProvider(tp)

	// Ativa a propagação do header 'traceparent' (W3C TraceContext) nas chamadas HTTP
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

