import { useMemo, useCallback } from 'react'
import {
  ReactFlow,
  type Node,
  type Edge,
  Background,
  Controls,
  MarkerType,
  type NodeTypes,
  Handle,
  Position,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import dagre from 'dagre'
import type { InfraGraph, GraphNode as ApiGraphNode } from '../api/client'

// --- Node color/icon config by kind ---

const kindConfig: Record<string, { bg: string; border: string; text: string; icon: string }> = {
  internet:  { bg: 'bg-blue-50',    border: 'border-blue-300',   text: 'text-blue-700',   icon: '🌐' },
  compute:   { bg: 'bg-purple-50',  border: 'border-purple-300', text: 'text-purple-700', icon: '⚡' },
  database:  { bg: 'bg-green-50',   border: 'border-green-300',  text: 'text-green-700',  icon: '🗄️' },
  cache:     { bg: 'bg-orange-50',  border: 'border-orange-300', text: 'text-orange-700', icon: '⚡' },
  queue:     { bg: 'bg-yellow-50',  border: 'border-yellow-300', text: 'text-yellow-700', icon: '📨' },
  storage:   { bg: 'bg-gray-50',    border: 'border-gray-300',   text: 'text-gray-700',   icon: '📦' },
  cdn:       { bg: 'bg-teal-50',    border: 'border-teal-300',   text: 'text-teal-700',   icon: '🌍' },
  network:   { bg: 'bg-red-50',     border: 'border-red-300',    text: 'text-red-700',    icon: '🔒' },
  secrets:   { bg: 'bg-amber-50',   border: 'border-amber-300',  text: 'text-amber-700',  icon: '🔐' },
  policy:    { bg: 'bg-indigo-50',  border: 'border-indigo-300', text: 'text-indigo-700', icon: '🛡️' },
}

function getKindConfig(kind: string) {
  return kindConfig[kind] || kindConfig.compute
}

// --- Custom Node Types ---

type InfraNodeData = {
  label: string
  kind: string
  service: string
}

type InfraNodeType = Node<InfraNodeData, 'infra'>

// --- Custom Node Component ---

function InfraNode({ data }: { data: InfraNodeData }) {
  const config = getKindConfig(data.kind)
  return (
    <div
      className={`px-4 py-3 rounded-lg border-2 shadow-sm min-w-[140px] text-center ${config.bg} ${config.border}`}
    >
      <Handle type="target" position={Position.Top} className="!bg-gray-400 !w-2 !h-2" />
      <div className="text-lg mb-1">{config.icon}</div>
      <div className={`text-sm font-semibold ${config.text}`}>{data.label}</div>
      {data.service && (
        <div className="text-xs text-gray-500 mt-0.5">{data.service}</div>
      )}
      <Handle type="source" position={Position.Bottom} className="!bg-gray-400 !w-2 !h-2" />
    </div>
  )
}

const nodeTypes: NodeTypes = {
  infra: InfraNode,
}

// --- Dagre layout ---

function getLayoutedElements(
  nodes: InfraNodeType[],
  edges: Edge[],
  direction: 'TB' | 'LR' = 'TB'
): { nodes: InfraNodeType[]; edges: Edge[] } {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: direction, nodesep: 60, ranksep: 80 })

  const nodeWidth = 180
  const nodeHeight = 80

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
  })

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  const layoutedNodes: InfraNodeType[] = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      },
    }
  })

  return { nodes: layoutedNodes, edges }
}

// --- Main Component ---

interface InfraGraphViewProps {
  graph: InfraGraph
}

export default function InfraGraphView({ graph }: InfraGraphViewProps) {
  const { nodes, edges } = useMemo(() => {
    const rfNodes: InfraNodeType[] = graph.nodes.map((n: ApiGraphNode) => ({
      id: n.id,
      type: 'infra',
      position: { x: 0, y: 0 },
      data: {
        label: n.label,
        kind: n.kind,
        service: n.service,
      },
    }))

    const rfEdges: Edge[] = graph.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      label: e.label,
      type: 'smoothstep',
      animated: true,
      style: { stroke: '#94a3b8', strokeWidth: 2 },
      labelStyle: { fontSize: 11, fill: '#64748b', fontWeight: 500 },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: '#94a3b8',
      },
    }))

    return getLayoutedElements(rfNodes, rfEdges)
  }, [graph])

  const onInit = useCallback(() => {
    // React Flow will fit the view on init
  }, [])

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      onInit={onInit}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      nodesDraggable={true}
      nodesConnectable={false}
      elementsSelectable={true}
      minZoom={0.3}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#e2e8f0" gap={20} />
      <Controls showInteractive={false} />
    </ReactFlow>
  )
}
