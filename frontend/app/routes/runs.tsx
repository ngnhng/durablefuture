export function meta() {
  return [
    { title: "Runs - DurableFuture" },
    { name: "description", content: "Task runs management" },
  ];
}

export default function Runs() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-semibold text-white mb-8">Runs</h1>
        <div className="bg-gray-900 rounded-lg p-8">
          <p className="text-gray-400">Task runs will be displayed here.</p>
        </div>
      </div>
    </div>
  );
}