# Prometheus en el cluster (kind) - pasos y contexto

Este README documenta los pasos usados para instalar Prometheus con Helm en el
namespace `dev`, aplicar el ServiceMonitor (`pthConfig.yml`) y comprobar que las
métricas se están recogiendo antes de configurar HPA.

## 1) Instalación con Helm (kube-prometheus-stack)

Objetivo: desplegar Prometheus y sus componentes dentro del cluster.

Comandos:

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kps prometheus-community/kube-prometheus-stack -n dev \
  --set prometheus.service.type=NodePort \
  --set prometheus.service.nodePort=30900
```

¿Qué hace?

- Crea recursos de Prometheus, Alertmanager y Grafana en `dev`.
- Instala CRDs necesarios para ServiceMonitor, PrometheusRule, etc.

## 2) Aplicar el ServiceMonitor (pthConfig.yml)

Objetivo: decirle a Prometheus que scrapee el puerto de métricas del Service de
la app (`staticsite-sample-svc`).

Comando:

```
kubectl apply -f prometheus/pthConfig.yml -n dev
```

¿Qué hace?

- Crea un `ServiceMonitor` en `dev`.
- Selecciona el Service con labels `app=staticsite` y `staticsite=staticsite-sample`.
- Scrapea el puerto `metrics` en la ruta `/metrics`.

## 3) Acceso a Prometheus

Objetivo: abrir la UI de Prometheus desde el host local (kind no expone LB).

Con NodePort (kind + `kind-config.yaml`):

- Abrir `http://localhost:9090`

Con port-forward (alternativa):

```
kubectl port-forward -n dev svc/kps-kube-prometheus-stack-prometheus 9090:9090
```

Uso:
- Ir a `Status -> Target Health` y comprobar que el ServiceMonitor aparece en verde.

## 4) Port-forward a las métricas de la app (opcional)

Objetivo: verificar que el exporter expone métricas desde el Service.

Comando:

```
kubectl port-forward -n dev svc/staticsite-sample-svc 9113:9113
```

Uso:

- Abrir `http://localhost:9113/metrics`.
- Deberías ver métricas `nginx_*`.

## 5) Ver métricas en la UI de Prometheus

Objetivo: confirmar que Prometheus recibe datos antes de configurar el HPA.

Ejemplo de consulta (peticiones por segundo):

```
sum by (pod) (rate(nginx_http_requests_total[1m]))
```

¿Si no aparecen resultados?

- Revisa `Status -> Target Health` en Prometheus.
- Confirma que `pthConfig.yml` está aplicado y el Service expone el puerto `metrics`.
 - Asegúrate de que estás usando el mismo namespace donde se desplegó la app.

## 6) Antes de crear el HPA

Checklist:

- La UI de Prometheus muestra el target `staticsite-sample` en verde.
- El endpoint `/metrics` responde en `http://localhost:9113/metrics`.
- La consulta `nginx_http_requests_total` devuelve series.

Cuando esto esté OK, se puede avanzar a configurar el HPA con métricas custom.
