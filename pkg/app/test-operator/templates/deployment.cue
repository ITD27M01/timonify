package templates

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	timoniv1 "timoni.sh/core/v1alpha1"
)

#Deployment: appsv1.#Deployment & {
	#config:    #ConfigapiVersion: apps/v1
kind:       Deployment
metadata: {
  name: #config.metadata.name + "-controller-manager"
  labels: #config.test-operator.metadata.labels & {
    {
    	"control-plane": "controller-manager"
    }
	spec: appsv1.#DeploymentSpec & {
replicas: #config.controllerManager.replicas
		selector: {
    {
    	matchLabels: {
    		"control-plane": "controller-manager"
    	}
    }
    {{- include "test-operator.selectorLabels" . | nindent 6 }}
		}
		template: {
			metadata: {
				labels: {
        {
        	"control-plane": "controller-manager"
        }
      {{- include "test-operator.selectorLabels" . | nindent 8 }}
			}
			spec: corev1.#PodSpec & {
      {
      	volumes: [{
      		configMap: {
      			name: "#config.metadata.name + \"-manager-config\""
      		}
      		name: "manager-config"
      	}, {
      		name: "secret-volume"
      		secret: {
      			secretName: "#config.metadata.name + \"-secret-ca\""
      		}
      	}]
      	topologySpreadConstraints: [{
      		matchLabelKeys: ["app", "pod-template-hash"]
      		maxSkew:           1
      		topologyKey:       "kubernetes.io/hostname"
      		whenUnsatisfiable: "DoNotSchedule"
      	}]
      	terminationGracePeriodSeconds: 10
      	serviceAccountName:            "#config.metadata.name + \"-controller-manager\""
      	nodeSelector:                  "{{- toYaml .Values.controllerManager.nodeSelector | nindent 8 }}"
      	imagePullSecrets: [{
      		name: "#config.metadata.name + \"-secret-registry-credentials\""
      	}]
      	containers: [{
      		args: "{{- toYaml .Values.controllerManager.kubeRbacProxy.args | nindent 8 }}"
      		env: [{
      			name:  "KUBERNETES_CLUSTER_DOMAIN"
      			value: "{{ quote .Values.kubernetesClusterDomain }}"
      		}]
      		image: "{{ .Values.controllerManager.kubeRbacProxy.image.repository }}:{{ .Values.controllerManager.kubeRbacProxy.image.tag | default .Chart.AppVersion }}"
      		name:  "kube-rbac-proxy"
      		ports: [{
      			containerPort: 8443
      			name:          "https"
      		}]
      		resources: {}
      	}, {
      		args: "{{- toYaml .Values.controllerManager.manager.args | nindent 8 }}"
      		command: ["/manager"]
      		env: [{
      			name: "VAR1"
      			valueFrom: {
      				secretKeyRef: {
      					key:  "VAR1"
      					name: "#config.metadata.name + \"-secret-vars\""
      				}
      			}
      		}, {
      			name:  "VAR2"
      			value: "{{ quote .Values.controllerManager.manager.env.var2 }}"
      		}, {
      			name:  "VAR3_MY_ENV"
      			value: "{{ quote .Values.controllerManager.manager.env.var3MyEnv }}"
      		}, {
      			name: "VAR4"
      			valueFrom: {
      				configMapKeyRef: {
      					key:  "VAR4"
      					name: "#config.metadata.name + \"-configmap-vars\""
      				}
      			}
      		}, {
      			name: "VAR5"
      			valueFrom: {
      				fieldRef: {
      					fieldPath: "metadata.namespace"
      				}
      			}
      		}, {
      			name: "VAR6"
      			valueFrom: {
      				resourceFieldRef: {
      					divisor:  "0"
      					resource: "limits.cpu"
      				}
      			}
      		}, {
      			name:  "KUBERNETES_CLUSTER_DOMAIN"
      			value: "{{ quote .Values.kubernetesClusterDomain }}"
      		}]
      		image:           "{{ .Values.controllerManager.manager.image.repository }}:{{ .Values.controllerManager.manager.image.tag | default .Chart.AppVersion }}"
      		imagePullPolicy: "{{ .Values.controllerManager.manager.imagePullPolicy }}"
      		livenessProbe: {
      			httpGet: {
      				path: "/healthz"
      				port: 8081
      			}
      			initialDelaySeconds: 15
      			periodSeconds:       20
      		}
      		name: "manager"
      		readinessProbe: {
      			httpGet: {
      				path: "/readyz"
      				port: 8081
      			}
      			initialDelaySeconds: 5
      			periodSeconds:       10
      		}
      		resources:       "{{- toYaml .Values.controllerManager.manager.resources | nindent 10 }}"
      		securityContext: "{{- toYaml .Values.controllerManager.manager.containerSecurityContext | nindent 10 }}"
      		volumeMounts: [{
      			mountPath: "/controller_manager_config.yaml"
      			name:      "manager-config"
      			subPath:   "controller_manager_config.yaml"
      		}, {
      			mountPath: "/my.ca"
      			name:      "secret-volume"
      		}]
      	}]
      	securityContext: {
      		runAsNonRoot: true
      	}
      }
			}
		}
	}
}