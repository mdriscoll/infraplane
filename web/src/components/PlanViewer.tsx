import type { InfrastructurePlan } from '../api/client'

export default function PlanViewer({ plan }: { plan: InfrastructurePlan }) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-gray-900">
          {plan.plan_type === 'hosting' ? 'Hosting Plan' : 'Migration Plan'}
        </h3>
        <span className="text-xs text-gray-500">
          {new Date(plan.created_at).toLocaleString()}
        </span>
      </div>

      {plan.from_provider && plan.to_provider && (
        <div className="mb-4 flex items-center gap-2 text-sm text-gray-600">
          <span className="uppercase font-medium">{plan.from_provider}</span>
          <span>&rarr;</span>
          <span className="uppercase font-medium">{plan.to_provider}</span>
        </div>
      )}

      {plan.estimated_cost && (
        <div className="mb-4 p-4 bg-indigo-50 rounded-lg">
          <p className="text-sm font-medium text-indigo-900">
            Estimated Cost: ${plan.estimated_cost.monthly_cost_usd.toFixed(2)}/month
          </p>
          {plan.estimated_cost.breakdown && (
            <div className="mt-2 grid grid-cols-2 gap-1 text-xs text-indigo-700">
              {Object.entries(plan.estimated_cost.breakdown).map(([key, value]) => (
                <div key={key}>
                  {key}: ${value.toFixed(2)}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="prose prose-sm max-w-none">
        <pre className="whitespace-pre-wrap text-sm bg-gray-50 p-4 rounded-lg border border-gray-100 overflow-auto">
          {plan.content}
        </pre>
      </div>
    </div>
  )
}
