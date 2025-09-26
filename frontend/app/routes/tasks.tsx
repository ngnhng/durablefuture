export function meta() {
  return [
    { title: "Tasks - DurableFuture" },
    { name: "description", content: "Task management" },
  ];
}

export default function Tasks() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-semibold text-white mb-8">Tasks</h1>
        <div className="bg-gray-900 rounded-lg p-8">
          <p className="text-gray-400">Your tasks will be displayed here once you complete the setup.</p>
        </div>
      </div>
    </div>
  );
}