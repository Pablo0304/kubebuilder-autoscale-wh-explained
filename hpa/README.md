# HPA (autoscaling) - guía simple y paso a paso

Objetivo: escalar el Deployment `staticsite-sample-web` en función de las
peticiones HTTP por pod, usando las métricas que expone el exporter de NGINX.

Este HPA usa **métricas custom** (no CPU/mem), por eso necesitas el
**Prometheus Adapter** para que Kubernetes pueda leer métricas desde Prometheus.

## 1) Requisitos previos (checklist)

Antes de aplicar el HPA, comprueba:

- El ServiceMonitor (`prometheus/pthConfig.yml`) está aplicado y el target está UP.
- En Prometheus existe la métrica `nginx_http_requests_total`.
- El Service `staticsite-sample-svc` expone el puerto `metrics` (9113).

Nota: usa el mismo namespace donde desplegaste todo (por defecto `dev`).

## 2) Instalar Prometheus Adapter

Si aún no lo tienes, se instala con Helm:

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prom-adapter prometheus-community/prometheus-adapter -n dev \
  --set prometheus.url=http://kps-kube-prometheus-stack-prometheus.dev.svc \
  --set prometheus.port=9090
```

## 3) Configurar el Adapter para la métrica de requests

El adapter necesita saber cómo convertir la métrica de Prometheus a una
métrica consumible por el HPA. Aplica este ConfigMap (el nombre debe ser
`prom-adapter-prometheus-adapter`):

```
kubectl apply -f hpa/adapter-config.yml -n dev
```

Después reinicia el adapter para que cargue la config (nombre del Deployment según el release):

```
kubectl rollout restart deployment/prom-adapter-prometheus-adapter -n dev
```

## 4) Aplicar el HPA

```
kubectl apply -f hpa/hpa.yml -n dev
```

## 5) Verificar

Ver HPA:

```
kubectl get hpa -n dev
```

Ver métricas disponibles:

```
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1" | grep nginx_http_requests
```

## 6) Qué está haciendo el HPA

El HPA usa la métrica `nginx_http_requests_per_second` (definida en el adapter) y
escala cuando la media por pod supera el valor objetivo.

Si quieres subir o bajar el umbral, cambia `averageValue` en `hpa.yml`.
