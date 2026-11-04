// Lets Node load Kibana's TypeScript sources as they are: their relative imports leave out the `.ts` extension.
import { registerHooks } from 'node:module';

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith('.') && !/\.[cm]?[jt]s$/.test(specifier)) {
      try {
        return nextResolve(`${specifier}.ts`, context);
      } catch {
        // Not a TypeScript sibling; resolve as written.
      }
    }
    return nextResolve(specifier, context);
  },
});
