import type { Store, StoreValue } from "nanostores";
import type { Accessor } from "solid-js";
import { onCleanup } from "solid-js";
import { createStore, reconcile } from "solid-js/store";

export function useStore<SomeStore extends Store, Value extends StoreValue<SomeStore>>(
  store: SomeStore,
): Accessor<Value> {
  // Activate the store so `get()` is populated, then hand off to the real
  // subscriber and unbind — avoids a dangling activation (https://github.com/nanostores/solid/issues/19).
  const unbindActivation = store.listen(() => {});

  const [state, setState] = createStore({ value: store.get() as Value });

  // `reconcile` diffs each update so only changed paths re-trigger (fine-grained).
  const unsubscribe = store.subscribe((value) => {
    setState("value", reconcile(value as Value));
  });

  onCleanup(() => unsubscribe());
  unbindActivation();

  return () => state.value;
}
