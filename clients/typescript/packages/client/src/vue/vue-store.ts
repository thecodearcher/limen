import type { Store, StoreValue } from "nanostores";
import type { DeepReadonly, ShallowRef, UnwrapNestedRefs } from "vue";
import { getCurrentScope, onScopeDispose, readonly, shallowRef } from "vue";

/** What {@link useStore} returns: a readonly reactive ref of the store value. */
export type ReactiveValue<Value> = DeepReadonly<UnwrapNestedRefs<ShallowRef<Value>>>;

/**
 * Subscribe a Vue component to a nanostores store.
 *
 * Modeled on `@nanostores/vue`'s `useStore`: a `shallowRef` mirrors the store
 * value, `subscribe` keeps it in sync (firing once immediately), and the
 * listener is torn down on scope dispose. On the server (no `window`) it reads
 * once without subscribing, so a store seeded with `initialSession` renders
 * without leaving a subscription behind.
 */
export function useStore<SomeStore extends Store>(store: SomeStore): ReactiveValue<StoreValue<SomeStore>> {
  type Value = StoreValue<SomeStore>;

  const state = shallowRef<Value>(store.get());

  if (typeof window !== "undefined") {
    const unsubscribe = store.subscribe((value) => {
      state.value = value;
    });
    if (getCurrentScope()) {
      onScopeDispose(unsubscribe);
    }
  }

  return readonly(state);
}
