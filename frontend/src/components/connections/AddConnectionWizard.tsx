"use client"

import React, { useState } from "react"
import { createCluster, getAgentManifest } from "@/lib/api"
import { ClusterConnection } from "@/types/cluster"

interface AddConnectionWizardProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (cluster: ClusterConnection) => void
}

export function AddConnectionWizard({ isOpen, onClose, onSuccess }: AddConnectionWizardProps) {
  const [step, setStep] = useState<number>(1)
  
  // Form State
  const [targetType, setTargetType] = useState<string>("k8s")
  const [provider, setProvider] = useState<string>("aws")
  const [connectionMode, setConnectionMode] = useState<string>("agent")
  const [name, setName] = useState<string>("")
  const [environment, setEnvironment] = useState<string>("production")
  const [endpoint, setEndpoint] = useState<string>("")
  const [bearerToken, setBearerToken] = useState<string>("")

  // Result state
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [createdCluster, setCreatedCluster] = useState<ClusterConnection | null>(null)
  const [agentCommand, setAgentCommand] = useState<string>("")

  if (!isOpen) return null

  const handleNext = () => {
    setError(null)
    if (step === 2 && !name.trim()) {
      setName(`${provider.toUpperCase()} Cluster`)
    }
    setStep(step + 1)
  }

  const handleCreate = async () => {
    setLoading(true)
    setError(null)

    try {
      let providerName = "AWS / EKS"
      let clusterType = "EKS"

      switch (provider) {
        case "local":
          providerName = "Local Development"
          clusterType = "Minikube / Kind"
          break
        case "onprem":
          providerName = "On-Premises"
          clusterType = "K3s / OpenShift"
          break
        case "aws":
          providerName = "AWS / EKS"
          clusterType = "EKS"
          break
        case "gcp":
          providerName = "Google Cloud / GKE"
          clusterType = "GKE"
          break
        case "azure":
          providerName = "Microsoft Azure / AKS"
          clusterType = "AKS"
          break
        default:
          providerName = "Custom Kubernetes"
          clusterType = "Conformant K8s"
      }

      const res = await createCluster({
        name: name || `${providerName} (${environment})`,
        environment,
        provider: providerName,
        clusterType,
        connectionMode: connectionMode as any,
        endpoint: endpoint.trim(),
        bearerToken: bearerToken.trim(),
      })

      setCreatedCluster(res.cluster)

      if (connectionMode === "agent") {
        const manifestData = await getAgentManifest(res.cluster.id)
        setAgentCommand(manifestData.command)
      }

      setStep(4)
    } catch (e: any) {
      setError(e.message || "Failed to create cluster connection")
    } finally {
      setLoading(false)
    }
  }

  const handleFinish = () => {
    if (createdCluster) {
      onSuccess(createdCluster)
    }
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl overflow-hidden rounded-2xl border border-zinc-800 bg-zinc-950 shadow-2xl">
        {/* Wizard Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 px-6 py-4">
          <div>
            <h2 className="text-lg font-semibold text-zinc-100">Add Infrastructure Connection</h2>
            <p className="text-xs text-zinc-500 mt-0.5">
              Step {step} of 4 — Connect local, remote, or cloud Kubernetes clusters
            </p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Step Progress Bar */}
        <div className="flex h-1 bg-zinc-900">
          <div
            className="bg-emerald-500 transition-all duration-300"
            style={{ width: `${(step / 4) * 100}%` }}
          />
        </div>

        {/* Modal Body */}
        <div className="p-6">
          {error && (
            <div className="mb-4 rounded-lg border border-rose-800/50 bg-rose-950/40 p-3 text-xs text-rose-300">
              {error}
            </div>
          )}

          {/* STEP 1: What are you connecting? */}
          {step === 1 && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-zinc-200">What are you connecting?</h3>
              <div className="grid grid-cols-2 gap-3">
                <button
                  onClick={() => setTargetType("k8s")}
                  className={`
                    flex items-center gap-3 rounded-xl border p-4 text-left transition-all
                    ${targetType === "k8s" ? "border-emerald-500 bg-emerald-950/20 text-zinc-100" : "border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:border-zinc-700"}
                  `}
                >
                  <div className="rounded-lg bg-emerald-500/10 p-2 text-emerald-400">
                    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                  </div>
                  <div>
                    <div className="font-semibold text-sm">Kubernetes Cluster</div>
                    <div className="text-[11px] text-zinc-500 mt-0.5">EKS, GKE, AKS, Minikube, Kind, On-Prem</div>
                  </div>
                </button>

                <div className="flex items-center gap-3 rounded-xl border border-zinc-900 bg-zinc-900/20 p-4 text-zinc-600 opacity-60 cursor-not-allowed">
                  <div className="rounded-lg bg-zinc-800 p-2 text-zinc-600">
                    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 001-9.999 5.002 5.002 0 00-9.78 2.096A4.001 4.001 0 003 15z" />
                    </svg>
                  </div>
                  <div>
                    <div className="font-semibold text-sm">Cloud Account</div>
                    <div className="text-[11px] text-zinc-600">Coming soon in Phase 7</div>
                  </div>
                </div>

                <div className="flex items-center gap-3 rounded-xl border border-zinc-900 bg-zinc-900/20 p-4 text-zinc-600 opacity-60 cursor-not-allowed">
                  <div className="rounded-lg bg-zinc-800 p-2 text-zinc-600">
                    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                    </svg>
                  </div>
                  <div>
                    <div className="font-semibold text-sm">VM / Host Server</div>
                    <div className="text-[11px] text-zinc-600">Coming soon</div>
                  </div>
                </div>

                <div className="flex items-center gap-3 rounded-xl border border-zinc-900 bg-zinc-900/20 p-4 text-zinc-600 opacity-60 cursor-not-allowed">
                  <div className="rounded-lg bg-zinc-800 p-2 text-zinc-600">
                    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                    </svg>
                  </div>
                  <div>
                    <div className="font-semibold text-sm">Observability Backend</div>
                    <div className="text-[11px] text-zinc-600">Prometheus / OpenTelemetry</div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* STEP 2: Where is it running? */}
          {step === 2 && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-zinc-200">Where is your Kubernetes cluster running?</h3>
              <div className="grid grid-cols-3 gap-3">
                {[
                  { id: "local", name: "Local Dev", sub: "Minikube / Kind / K3d" },
                  { id: "onprem", name: "On-Premises", sub: "Bare-Metal / OpenShift" },
                  { id: "aws", name: "AWS EKS", sub: "Amazon Elastic K8s" },
                  { id: "gcp", name: "Google GKE", sub: "Google Kubernetes Engine" },
                  { id: "azure", name: "Azure AKS", sub: "Azure Kubernetes Service" },
                  { id: "other", name: "Other Cloud", sub: "Arbitrary Conformant K8s" },
                ].map((item) => (
                  <button
                    key={item.id}
                    onClick={() => setProvider(item.id)}
                    className={`
                      rounded-xl border p-3 text-left transition-all
                      ${provider === item.id ? "border-emerald-500 bg-emerald-950/20 text-zinc-100" : "border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:border-zinc-700"}
                    `}
                  >
                    <div className="font-semibold text-xs text-zinc-200">{item.name}</div>
                    <div className="text-[10px] text-zinc-500 mt-0.5">{item.sub}</div>
                  </button>
                ))}
              </div>

              <div className="grid grid-cols-2 gap-3 mt-4 pt-4 border-t border-zinc-900">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Connection Name</label>
                  <input
                    type="text"
                    placeholder="e.g. Production EKS"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 focus:border-emerald-500 focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Environment</label>
                  <select
                    value={environment}
                    onChange={(e) => setEnvironment(e.target.value)}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 focus:border-emerald-500 focus:outline-none"
                  >
                    <option value="production">Production</option>
                    <option value="staging">Staging</option>
                    <option value="development">Development</option>
                    <option value="on-premises">On-Premises</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {/* STEP 3: Connection method */}
          {step === 3 && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-zinc-200">Select Connection Method</h3>
              <div className="space-y-2">
                <button
                  onClick={() => setConnectionMode("agent")}
                  className={`
                    w-full flex items-start gap-3 rounded-xl border p-3.5 text-left transition-all
                    ${connectionMode === "agent" ? "border-emerald-500 bg-emerald-950/20 text-zinc-100" : "border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:border-zinc-700"}
                  `}
                >
                  <span className="mt-0.5 rounded bg-emerald-500/10 p-1.5 text-emerald-400">
                    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                  </span>
                  <div>
                    <div className="font-semibold text-xs text-zinc-100 flex items-center gap-2">
                      Garund Secure Agent (Recommended)
                      <span className="rounded bg-emerald-900/40 text-emerald-400 text-[10px] px-2 py-0.5 font-normal">No public API exposed</span>
                    </div>
                    <p className="text-[11px] text-zinc-500 mt-0.5">
                      Lightweight in-cluster agent establishes outbound TLS connection to Garund. Ideal for private & on-prem clusters.
                    </p>
                  </div>
                </button>

                <button
                  onClick={() => setConnectionMode("local_kubeconfig")}
                  className={`
                    w-full flex items-start gap-3 rounded-xl border p-3.5 text-left transition-all
                    ${connectionMode === "local_kubeconfig" ? "border-emerald-500 bg-emerald-950/20 text-zinc-100" : "border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:border-zinc-700"}
                  `}
                >
                  <span className="mt-0.5 rounded bg-blue-500/10 p-1.5 text-blue-400">
                    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </span>
                  <div>
                    <div className="font-semibold text-xs text-zinc-100">Local Kubeconfig Context</div>
                    <p className="text-[11px] text-zinc-500 mt-0.5">
                      Uses local <code className="text-zinc-400">~/.kube/config</code> context directly on the server host.
                    </p>
                  </div>
                </button>

                <button
                  onClick={() => setConnectionMode("service_account_token")}
                  className={`
                    w-full flex items-start gap-3 rounded-xl border p-3.5 text-left transition-all
                    ${connectionMode === "service_account_token" ? "border-emerald-500 bg-emerald-950/20 text-zinc-100" : "border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:border-zinc-700"}
                  `}
                >
                  <span className="mt-0.5 rounded bg-amber-500/10 p-1.5 text-amber-400">
                    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 0121 9z" />
                    </svg>
                  </span>
                  <div>
                    <div className="font-semibold text-xs text-zinc-100">Scoped ServiceAccount Token</div>
                    <p className="text-[11px] text-zinc-500 mt-0.5">
                      Short-lived Kubernetes ServiceAccount token with read-only RBAC permissions.
                    </p>
                  </div>
                </button>
              </div>

              {connectionMode === "service_account_token" && (
                <div className="mt-3 space-y-3 pt-3 border-t border-zinc-900">
                  <div>
                    <label className="block text-xs font-medium text-zinc-400 mb-1">API Server Endpoint URL</label>
                    <input
                      type="text"
                      placeholder="https://eks.amazonaws.com:443"
                      value={endpoint}
                      onChange={(e) => setEndpoint(e.target.value)}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 focus:border-emerald-500 focus:outline-none font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-zinc-400 mb-1">Scoped ServiceAccount Bearer Token</label>
                    <input
                      type="password"
                      placeholder="eyJhbGciOiJSUzI1NiIs..."
                      value={bearerToken}
                      onChange={(e) => setBearerToken(e.target.value)}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 focus:border-emerald-500 focus:outline-none font-mono"
                    />
                  </div>
                </div>
              )}

              {/* Security Warning Notice */}
              <div className="rounded-xl border border-amber-900/40 bg-amber-950/20 p-3 flex items-start gap-2 text-[11px] text-amber-300">
                <svg className="h-4 w-4 shrink-0 text-amber-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span>
                  <strong>Security Guarantee:</strong> Garund NEVER logs, exposes, or saves unrestricted admin credentials to frontend storage. Connections utilize scoped RBAC roles.
                </span>
              </div>
            </div>
          )}

          {/* STEP 4: Installation Instructions */}
          {step === 4 && createdCluster && (
            <div className="space-y-4">
              <div className="rounded-xl border border-emerald-800/40 bg-emerald-950/20 p-4 flex items-center gap-3 text-emerald-300">
                <span className="h-3 w-3 rounded-full bg-emerald-500 animate-pulse" />
                <div>
                  <div className="font-semibold text-xs">Connection Registered: {createdCluster.name}</div>
                  <div className="text-[11px] text-emerald-400/80">
                    Mode: {createdCluster.connectionMode} • Status: {createdCluster.status}
                  </div>
                </div>
              </div>

              {connectionMode === "agent" && agentCommand && (
                <div className="space-y-2">
                  <h4 className="text-xs font-medium text-zinc-300">Run this command in your Kubernetes cluster to deploy the agent:</h4>
                  <div className="relative rounded-xl border border-zinc-800 bg-zinc-900/90 p-3 font-mono text-[11px] text-emerald-400 overflow-x-auto max-h-48">
                    <pre>{agentCommand}</pre>
                  </div>
                  <p className="text-[10px] text-zinc-500">
                    This deploys a lightweight ServiceAccount and Deployment in <code className="text-zinc-400">garund-system</code>.
                  </p>
                </div>
              )}

              {connectionMode !== "agent" && (
                <div className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 text-xs text-zinc-300">
                  Your cluster connection has been saved and is now accessible via the global cluster switcher.
                </div>
              )}
            </div>
          )}
        </div>

        {/* Wizard Footer Controls */}
        <div className="flex items-center justify-between border-t border-zinc-800 px-6 py-4 bg-zinc-900/30">
          {step > 1 && step < 4 ? (
            <button
              onClick={() => setStep(step - 1)}
              className="rounded-lg border border-zinc-800 px-4 py-2 text-xs font-medium text-zinc-300 hover:bg-zinc-800"
            >
              Back
            </button>
          ) : (
            <div />
          )}

          {step < 3 && (
            <button
              onClick={handleNext}
              className="rounded-lg bg-emerald-600 px-5 py-2 text-xs font-medium text-white shadow-lg hover:bg-emerald-500 transition-colors"
            >
              Continue
            </button>
          )}

          {step === 3 && (
            <button
              onClick={handleCreate}
              disabled={loading}
              className="rounded-lg bg-emerald-600 px-5 py-2 text-xs font-medium text-white shadow-lg hover:bg-emerald-500 disabled:opacity-50 transition-colors"
            >
              {loading ? "Registering..." : "Connect Cluster"}
            </button>
          )}

          {step === 4 && (
            <button
              onClick={handleFinish}
              className="rounded-lg bg-emerald-600 px-6 py-2 text-xs font-medium text-white shadow-lg hover:bg-emerald-500 transition-colors"
            >
              Done & View Dashboard
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
