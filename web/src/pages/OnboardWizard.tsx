import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useOnboardApplication, useComplianceFrameworks } from '../hooks/useApi'
import { pickAndReadDirectory, isDirectoryPickerSupported } from '../lib/directoryPicker'
import type { FileContent } from '../lib/directoryPicker'
import type { OnboardResult } from '../api/client'
import ResourceList from '../components/ResourceList'
import PlanViewer from '../components/PlanViewer'
import Spinner from '../components/Spinner'

type WizardStep = 'provider' | 'source' | 'processing' | 'results'

const STEPS: { key: WizardStep; label: string }[] = [
  { key: 'provider', label: 'Provider' },
  { key: 'source', label: 'Your Code' },
  { key: 'processing', label: 'Analyzing' },
  { key: 'results', label: 'Results' },
]

const PROGRESS_MESSAGES = [
  'Registering your application...',
  'Scanning your codebase for infrastructure...',
  'Detecting databases, caches, and services...',
  'Generating your hosting plan...',
  'Estimating costs...',
]

function StepIndicator({ currentStep }: { currentStep: WizardStep }) {
  const currentIdx = STEPS.findIndex((s) => s.key === currentStep)

  return (
    <div className="flex items-center justify-center gap-1 mb-10">
      {STEPS.map((step, i) => {
        const isCompleted = i < currentIdx
        const isCurrent = i === currentIdx
        return (
          <div key={step.key} className="flex items-center">
            {i > 0 && (
              <div className={`w-10 h-0.5 mx-1 ${isCompleted ? 'bg-indigo-500' : 'bg-gray-200'}`} />
            )}
            <div className="flex flex-col items-center gap-1">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold transition-colors ${
                  isCompleted
                    ? 'bg-indigo-500 text-white'
                    : isCurrent
                      ? 'bg-indigo-100 text-indigo-700 ring-2 ring-indigo-500'
                      : 'bg-gray-100 text-gray-400'
                }`}
              >
                {isCompleted ? (
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                  </svg>
                ) : (
                  i + 1
                )}
              </div>
              <span className={`text-xs ${isCurrent ? 'text-indigo-700 font-medium' : 'text-gray-400'}`}>
                {step.label}
              </span>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default function OnboardWizard() {
  const navigate = useNavigate()
  const onboard = useOnboardApplication()

  const [step, setStep] = useState<WizardStep>('provider')
  const [provider, setProvider] = useState<'aws' | 'gcp' | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [pickedFiles, setPickedFiles] = useState<FileContent[] | null>(null)
  const [progressIdx, setProgressIdx] = useState(0)
  const [result, setResult] = useState<OnboardResult | null>(null)
  const [selectedFrameworks, setSelectedFrameworks] = useState<string[]>([])

  // Fetch compliance frameworks for the selected provider
  const { data: frameworks } = useComplianceFrameworks(provider || undefined)

  const toggleFramework = (id: string) => {
    setSelectedFrameworks((prev) =>
      prev.includes(id) ? prev.filter((f) => f !== id) : [...prev, id]
    )
  }

  // Cycle progress messages during processing
  useEffect(() => {
    if (step !== 'processing') return
    const interval = setInterval(() => {
      setProgressIdx((prev) => (prev + 1) % PROGRESS_MESSAGES.length)
    }, 4000)
    return () => clearInterval(interval)
  }, [step])

  const handleBrowse = async () => {
    try {
      const files = await pickAndReadDirectory()
      if (files && files.length > 0) {
        setPickedFiles(files)
      }
    } catch (err) {
      console.error('Failed to read directory:', err)
    }
  }

  const handleSubmit = () => {
    if (!provider || !name.trim()) return

    setStep('processing')
    setProgressIdx(0)

    onboard.mutate(
      {
        name: name.trim(),
        description: description.trim() || undefined,
        provider,
        compliance_frameworks: selectedFrameworks.length > 0 ? selectedFrameworks : undefined,
        files: pickedFiles || undefined,
      },
      {
        onSuccess: (data) => {
          setResult(data)
          setStep('results')
        },
        onError: () => {
          // Stay on processing step — error UI shown there
        },
      },
    )
  }

  const handleStartOver = () => {
    setStep('provider')
    setProvider(null)
    setName('')
    setDescription('')
    setPickedFiles(null)
    setResult(null)
    setSelectedFrameworks([])
    onboard.reset()
  }

  return (
    <div className="max-w-3xl mx-auto py-10 px-4">
      <div className="text-center mb-2">
        <Link to="/" className="text-sm text-gray-400 hover:text-gray-600">
          &larr; Back to Applications
        </Link>
      </div>

      <h1 className="text-2xl font-bold text-gray-900 text-center mb-1">
        {step === 'results' ? 'Your Hosting Plan' : 'Get Started with Infraplane'}
      </h1>
      {step !== 'results' && (
        <p className="text-gray-500 text-center mb-6">
          Point us at your code and we&apos;ll generate a cloud hosting plan.
        </p>
      )}

      <StepIndicator currentStep={step} />

      {/* Step 1: Provider Selection */}
      {step === 'provider' && (
        <div className="space-y-6">
          <h2 className="text-lg font-semibold text-gray-900 text-center">
            Where do you want to host?
          </h2>
          <div className="grid grid-cols-2 gap-4">
            {(['gcp', 'aws'] as const).map((p) => {
              const isSelected = provider === p
              const label = p === 'gcp' ? 'Google Cloud' : 'Amazon Web Services'
              const abbr = p.toUpperCase()
              return (
                <button
                  key={p}
                  onClick={() => setProvider(p)}
                  className={`relative p-6 rounded-xl border-2 text-left transition-all ${
                    isSelected
                      ? 'border-indigo-500 bg-indigo-50 shadow-md'
                      : 'border-gray-200 bg-white hover:border-gray-300 hover:shadow-sm'
                  }`}
                >
                  {isSelected && (
                    <div className="absolute top-3 right-3 w-6 h-6 bg-indigo-500 rounded-full flex items-center justify-center">
                      <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                      </svg>
                    </div>
                  )}
                  <div className={`text-2xl font-bold mb-1 ${isSelected ? 'text-indigo-700' : 'text-gray-900'}`}>
                    {abbr}
                  </div>
                  <div className={`text-sm ${isSelected ? 'text-indigo-600' : 'text-gray-500'}`}>
                    {label}
                  </div>
                </button>
              )
            })}
          </div>
          {/* Compliance frameworks (shown after provider selected) */}
          {provider && frameworks && frameworks.length > 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-medium text-gray-700">
                Compliance Frameworks <span className="text-gray-400">(optional)</span>
              </h3>
              <p className="text-xs text-gray-400">
                Select frameworks to enforce on generated infrastructure.
              </p>
              <div className="space-y-2">
                {frameworks.map((fw) => {
                  const isChecked = selectedFrameworks.includes(fw.id)
                  return (
                    <label
                      key={fw.id}
                      className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-all ${
                        isChecked
                          ? 'border-indigo-300 bg-indigo-50'
                          : 'border-gray-200 bg-white hover:border-gray-300'
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleFramework(fw.id)}
                        className="mt-0.5 h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                      />
                      <div>
                        <div className="text-sm font-medium text-gray-900">
                          {fw.name} {fw.version}
                        </div>
                        <div className="text-xs text-gray-500">{fw.description}</div>
                      </div>
                    </label>
                  )
                })}
              </div>
            </div>
          )}

          <div className="flex justify-end">
            <button
              onClick={() => setStep('source')}
              disabled={!provider}
              className="bg-indigo-600 text-white px-6 py-2.5 rounded-lg hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors font-medium"
            >
              Next
            </button>
          </div>
        </div>
      )}

      {/* Step 2: Code Source */}
      {step === 'source' && (
        <div className="space-y-5">
          <div>
            <label htmlFor="app-name" className="block text-sm font-medium text-gray-700 mb-1">
              Application Name
            </label>
            <input
              id="app-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. my-web-app"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              autoFocus
            />
          </div>

          <div>
            <label htmlFor="app-desc" className="block text-sm font-medium text-gray-700 mb-1">
              Description <span className="text-gray-400">(optional)</span>
            </label>
            <input
              id="app-desc"
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of your app"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Your Code
            </label>
            {isDirectoryPickerSupported() ? (
              <div className="space-y-2">
                <button
                  onClick={handleBrowse}
                  className="inline-flex items-center gap-2 text-sm bg-gray-100 text-gray-700 px-4 py-2.5 rounded-lg hover:bg-gray-200 transition-colors"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                  {pickedFiles ? 'Change Folder' : 'Browse Folder'}
                </button>
                {pickedFiles && (
                  <p className="text-sm text-green-600 flex items-center gap-1.5">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {pickedFiles.length} infrastructure file{pickedFiles.length !== 1 ? 's' : ''} detected
                  </p>
                )}
              </div>
            ) : (
              <p className="text-sm text-gray-400 italic">
                Folder browsing requires Chrome or Edge. Register via the applications page instead.
              </p>
            )}
          </div>

          <div className="flex justify-between pt-2">
            <button
              onClick={() => setStep('provider')}
              className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2.5 rounded-lg hover:bg-gray-100 transition-colors"
            >
              &larr; Back
            </button>
            <button
              onClick={handleSubmit}
              disabled={!name.trim() || !pickedFiles}
              className="bg-indigo-600 text-white px-6 py-2.5 rounded-lg hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors font-medium"
            >
              Generate My Hosting Plan
            </button>
          </div>
        </div>
      )}

      {/* Step 3: Processing */}
      {step === 'processing' && (
        <div className="text-center py-12">
          {onboard.isError ? (
            <div className="space-y-4">
              <div className="w-12 h-12 mx-auto rounded-full bg-red-100 flex items-center justify-center">
                <svg className="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900">Something went wrong</h3>
              <p className="text-sm text-red-500">{onboard.error?.message}</p>
              <div className="flex justify-center gap-3 pt-2">
                <button
                  onClick={() => setStep('source')}
                  className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-100 transition-colors"
                >
                  &larr; Go Back
                </button>
                <button
                  onClick={handleSubmit}
                  className="text-sm bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
                >
                  Try Again
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              <Spinner className="mx-auto text-indigo-600 h-10 w-10" />
              <div>
                <p className="text-lg font-medium text-gray-900">{PROGRESS_MESSAGES[progressIdx]}</p>
                <p className="text-sm text-gray-400 mt-2">This usually takes 30-60 seconds.</p>
              </div>
              <div className="w-48 mx-auto bg-gray-200 rounded-full h-1.5 overflow-hidden">
                <div className="bg-indigo-500 h-full rounded-full animate-pulse" style={{ width: '60%' }} />
              </div>
            </div>
          )}
        </div>
      )}

      {/* Step 4: Results */}
      {step === 'results' && result && (
        <div className="space-y-8">
          {/* Success header */}
          <div className="text-center">
            <div className="w-14 h-14 mx-auto mb-3 rounded-full bg-green-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <div className="flex items-center justify-center gap-2 mb-1">
              <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700">
                {result.application.provider.toUpperCase()}
              </span>
              {result.application.compliance_frameworks?.map((fw) => (
                <span key={fw} className="text-xs font-medium px-2 py-0.5 rounded-full bg-green-50 text-green-700">
                  {fw}
                </span>
              ))}
              <h2 className="text-xl font-bold text-gray-900">{result.application.name}</h2>
            </div>
            {result.application.description && (
              <p className="text-sm text-gray-500">{result.application.description}</p>
            )}
          </div>

          {/* Detected Resources */}
          {result.resources && result.resources.length > 0 && (
            <section>
              <h3 className="text-lg font-semibold text-gray-900 mb-3">
                Detected Resources ({result.resources.length})
              </h3>
              <ResourceList resources={result.resources} />
            </section>
          )}

          {/* Hosting Plan */}
          {result.plan && result.plan.id && result.plan.id !== '00000000-0000-0000-0000-000000000000' && result.plan.content && (
            <section>
              <h3 className="text-lg font-semibold text-gray-900 mb-3">Hosting Plan</h3>
              <PlanViewer plan={result.plan} />
            </section>
          )}

          {(!result.plan?.content || result.plan.id === '00000000-0000-0000-0000-000000000000') && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-center">
              <p className="text-sm text-yellow-800">
                Plan generation didn&apos;t complete. You can generate it from the application detail page.
              </p>
            </div>
          )}

          {/* CTAs */}
          <div className="flex justify-center gap-3 pt-2">
            <button
              onClick={handleStartOver}
              className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2.5 rounded-lg hover:bg-gray-100 transition-colors"
            >
              Start Over
            </button>
            <button
              onClick={() => navigate(`/apps/${result.application.name}`)}
              className="bg-indigo-600 text-white px-6 py-2.5 rounded-lg hover:bg-indigo-700 transition-colors font-medium"
            >
              View Application &rarr;
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
