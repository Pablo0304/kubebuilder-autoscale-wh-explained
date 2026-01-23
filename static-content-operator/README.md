# Static Content Operator (Kubebuilder) - Guía Completa

## Propósito

Este repositorio contiene un operador de Kubernetes construido con Kubebuilder.
El objetivo es definir un recurso personalizado (CRD) que despliegue un sitio
web estático en el cluster.

Este proyecto sirve como ejemplo sencillo de:

- Un único CRD para un sitio web.
- Controlador que crea Deployment + Service + ConfigMap.
- Uso básico de CRDs, RBAC y kustomize.

## Conceptos clave (resumen rápido)

- CRD (CustomResourceDefinition): define un tipo nuevo en Kubernetes.
- CR (CustomResource): instancia concreta de ese tipo (tu YAML).
- Controller/Reconciler: código que observa CRs y crea recursos reales.
- Group/Version/Kind (GVK): nombre canónico del tipo en Kubernetes.

## Arquitectura del ejemplo

Se usa un solo CRD `StaticSite` en el grupo `static.static.com`:

- `StaticSite` describe un sitio web con:
  - Deployment (pods que sirven la web)
  - Service (balanceo interno)

El controlador:

- Lee el StaticSite
- Crea ConfigMap (config de nginx) + Deployment + Service
- Añade un sidecar nginx-prometheus-exporter para métricas Prometheus

## Requisitos

- Go instalado (versión recomendada por Kubebuilder)
- kubebuilder instalado
- Docker instalado (para kind)
- kind instalado
- kubectl instalado

## Estructura relevante del repositorio

- `api/v1/`
  - Tipos Go para el CRD StaticSite
- `internal/controller/`
  - Lógica de reconciliación
- `config/`
  - Manifiestos kustomize (CRDs, RBAC, manager)
- `config/samples/`
  - Ejemplos de CRs para pruebas

## Pasos para inicializar (orden recomendado)

1. Inicializar proyecto Kubebuilder:

   - `kubebuilder init --domain static.com --repo staticwebsite`

2. Crear CRD StaticSite:

   - `kubebuilder create api --group static --version v1 --kind StaticSite`

3. Editar la lógica del controller:

   - Archivo: `internal/controller/staticsite_controller.go`
   - Objetivo: crear ConfigMap + Deployment + Service (web + métricas)

4. Generar manifiestos (CRDs/RBAC):

  - `make manifests`

5. Crear un cluster local con kind:

   - `kind create cluster --name kubebuilder`

6. Instalar CRDs en el cluster:

   - `make install`

7. Ejecutar el controlador local:

  - `make run`

8. Aplicar ejemplo (en namespace `dev`):

  - `kubectl apply -f config/samples/static_v1_staticsite.yaml -n dev`

## Ejecución local vs despliegue en el cluster

- `make run`: ejecuta el controlador en tu máquina. Es ideal para desarrollo local, pero se detiene si cierras la terminal.
- `make deploy`: instala el operador dentro del cluster (Deployment). Es el modo recomendado para entornos reales.

Flujo típico:

1. `make install` (instala CRDs)
2. `make deploy` (despliega el operador en el cluster)
3. `make undeploy` (elimina el operador del cluster)

## Definición de CRD (explicación)

Archivo: `api/v1/staticsite_types.go`

Campos principales:

- `spec.image`: imagen del contenedor que sirve la web
- `spec.port`: puerto del contenedor
- `spec.service.type`: ClusterIP / NodePort / LoadBalancer
- `spec.service.port`: puerto del Service

El controller añade:

- ConfigMap con `stub_status` en el puerto 8081 (solo localhost).
- Sidecar `nginx/nginx-prometheus-exporter:latest` en el puerto 9113.

## Ejemplo (sample)

Los ejemplos viven en `config/samples/` y se aplican con:

- `kubectl apply -f config/samples/static_v1_staticsite.yaml -n dev`

Ejemplo de StaticSite:

```yaml
apiVersion: static.static.com/v1
kind: StaticSite
metadata:
  name: staticsite-sample
spec:
  image: nginx:latest
  port: 80
  service:
    type: ClusterIP
    port: 80
```

## ¿Cómo comprobar que funciona?

1. Ver el StaticSite:

- `kubectl get staticsites -n dev`

2. Ver recursos creados:

- `kubectl get deployment -n dev`
- `kubectl get svc -n dev`
- `kubectl get pods -n dev`

3. Acceso a la web:

- Con NodePort (kind + `kind-config.yaml`):
  - Abrir `http://localhost:8080` (NodePort 30080)
- Con port-forward (alternativa):
  - `kubectl port-forward -n dev svc/staticsite-sample-svc 8080:80`

4. Métricas Prometheus (nginx exporter):

- Con NodePort (kind + `kind-config.yaml`):
  - Abrir `http://localhost:9113/metrics` (NodePort 30913)
- Con port-forward (alternativa):
  - `kubectl port-forward -n dev svc/staticsite-sample-svc 9113:9113`

5. Prometheus UI:

- Con NodePort (kind + `kind-config.yaml`):
  - Abrir `http://localhost:9090` (NodePort 30900)
- Con port-forward (alternativa):
  - `kubectl port-forward -n dev svc/kps-kube-prometheus-stack-prometheus 9090:9090`

## Buenas prácticas aplicadas

- Un operador por dominio funcional (static web).
- Un único CRD para un sitio.
- RBAC mínimo necesario.
- Kustomize para aplicar recursos agrupados.

## Troubleshooting rápido

- "no matches for kind ...": falta instalar CRD -> ejecutar `make install`.
- "ImagePullBackOff": imagen mal escrita o tag inexistente.
- "configmaps is forbidden": falta RBAC -> ejecutar `make manifests` y `make deploy`.

## Siguientes pasos recomendados

- HPA con métricas custom vía Prometheus Adapter.
- Añadir `status` con URL y estado.
- Validaciones OpenAPI en el CRD.
