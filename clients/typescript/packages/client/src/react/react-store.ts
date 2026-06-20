import type { Store, StoreValue } from "nanostores";
import { useCallback, useRef, useSyncExternalStore } from "react";

export function useStore<SomeStore extends Store>(store: SomeStore): StoreValue<SomeStore> {
  type Value = StoreValue<SomeStore>;

  const snapshotRef = useRef<Value | undefined>(store.get());

  const subscribe = useCallback(
    (onChange: () => void) => {
      const emitValue = (value: Value): void => {
        if (snapshotRef.current === value) {
          return;
        }
        snapshotRef.current = value;
        onChange();
      };
      // Resync before listening: the value may have changed between render and
      // this effect, and `listen` does not fire for the current value.
      emitValue(store.value);
      return store.listen(emitValue);
    },
    [store],
  );

  const get = (): Value => snapshotRef.current as Value;

  return useSyncExternalStore(subscribe, get, get);
}
