import { useEffect, useRef } from 'react'
import type { DeploymentEvent } from '../api/client'
import Spinner from './Spinner'

interface DeployLogProps {
  events: DeploymentEvent[]
  isStreaming: boolean
  isComplete: boolean
  finalStatus: 'succeeded' | 'failed' | null
}

const stepLabels: Record<string, string> = {
  initializing: 'Initializing',
  generating_terraform: 'Generating Terraform',
  validating: 'Validating',
  applying: 'Applying',
}

const steps = ['initializing', 'generating_terraform', 'validating', 'applying'] as const

export default function DeployLog({ events, isStreaming, isComplete, finalStatus }: DeployLogProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom on new events
  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [events.length])

  // Determine which steps are completed and which is active
  const completedSteps = new Set<string>()
  const currentStep = events.length > 0 ? events[events.length - 1].step : null

  for (const event of events) {
    for (const s of steps) {
      if (s === event.step) break
      completedSteps.add(s)
    }
  }
  if (isComplete && finalStatus === 'succeeded') {
    steps.forEach((s) => completedSteps.add(s))
  }

  return (
    <div className="space-y-4">
      {/* Step progress bar */}
      <div className="flex gap-4">
        {steps.map((step) => {
          const isDone = completedSteps.has(step)
          const isActive = currentStep === step && isStreaming
          return (
            <div key={step} className="flex items-center gap-1.5 text-xs">
              {isDone ? (
                <span className="text-green-500">{'\u2713'}</span>
              ) : isActive ? (
                <Spinner className="text-indigo-500" />
              ) : (
                <span className="text-gray-300">{'\u25CB'}</span>
              )}
              <span className={isDone ? 'text-green-700' : isActive ? 'text-indigo-700 font-medium' : 'text-gray-400'}>
                {stepLabels[step]}
              </span>
            </div>
          )
        })}
      </div>

      {/* Terminal log output */}
      <div className="bg-gray-900 rounded-lg p-4 max-h-96 overflow-y-auto font-mono text-sm">
        {events.map((event, i) => (
          <div key={i} className="flex gap-2 leading-relaxed">
            <span className="text-gray-500 select-none shrink-0">
              {new Date(event.timestamp).toLocaleTimeString()}
            </span>
            <span className={
              event.step === 'failed' ? 'text-red-400' :
              event.step === 'complete' ? 'text-green-400' :
              'text-gray-200'
            }>
              {event.message}
            </span>
          </div>
        ))}
        {isStreaming && events.length === 0 && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner className="text-gray-400" />
            <span>Waiting for output...</span>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Final status banner */}
      {isComplete && (
        <div className={`p-3 rounded-lg text-sm font-medium ${
          finalStatus === 'succeeded'
            ? 'bg-green-50 text-green-800 border border-green-200'
            : 'bg-red-50 text-red-800 border border-red-200'
        }`}>
          {finalStatus === 'succeeded'
            ? 'Deployment completed successfully.'
            : 'Deployment failed. Check the logs above for details.'}
        </div>
      )}
    </div>
  )
}
