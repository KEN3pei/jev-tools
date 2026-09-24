export const DEFAULT_MODEL = "jev-latest";
export const DEFAULT_BASE_URL = "https://api.typesafe.ai/v1/systemone";

export const QUESTIONS = {
  requested_level: {
    type: "choice",
    instructions:
      "What is the primary abstraction level requested by `user_request`, interpreted with `preceding_context`?",
    criteria: {
      conceptual_orientation:
        "A basic definition, orientation, or mental model is primary.",
      architecture_and_design_space:
        "System patterns, components, boundaries, alternatives, and tradeoffs are primary.",
      implementation_mechanics:
        "Concrete code, APIs, configuration, commands, or integration steps are primary.",
      operations_and_governance:
        "Deployment, monitoring, reliability, security, governance, or organizational operation is primary.",
      unclear: "The intended abstraction level cannot be inferred reliably."
    }
  },
  answer_entry_level: {
    type: "choice",
    instructions:
      "At what abstraction level does `candidate_answer` begin its substantive explanation?",
    criteria: {
      conceptual_orientation:
        "It first establishes a definition, orientation, or basic mental model.",
      architecture_and_design_space:
        "It first explains system patterns, components, relationships, alternatives, or tradeoffs.",
      implementation_mechanics:
        "It quickly enters code, APIs, products, configuration, commands, or integration mechanisms.",
      operations_and_governance:
        "It begins with deployment, monitoring, reliability, security, governance, or organizational operation.",
      unclear: "The answer's entry level cannot be inferred reliably."
    }
  },
  abstraction_mismatch: {
    type: "noul",
    instructions:
      "Does `candidate_answer` begin at a materially different abstraction level from the one primarily requested, making orientation harder even if later content is relevant?",
    criteria: {
      true: "The answer starts too concretely or too abstractly relative to the request.",
      false:
        "The answer starts at an appropriate level and then moves through detail coherently."
    }
  },
  premature_specificity: {
    type: "noul",
    instructions:
      "Does `candidate_answer` introduce code, products, APIs, configuration, or implementation mechanisms before establishing the mental model requested by the user?",
    criteria: {
      true:
        "Specific detail arrives before the reader has an adequate conceptual map.",
      false:
        "The conceptual map is established first, or implementation detail is clearly the primary request."
    }
  },
  progressive_disclosure: {
    type: "noul",
    instructions:
      "Does `candidate_answer` progress in an appropriate order from the requested level toward patterns, tradeoffs, examples, and implementation details?",
    criteria: {
      true:
        "The answer introduces detail in an order that supports understanding.",
      false:
        "The answer jumps between levels or introduces lower-level detail before the necessary foundation."
    }
  },
  prerequisite_fit: {
    type: "noul",
    instructions:
      "Does `candidate_answer` avoid assuming that the user already understands concepts they are currently asking to understand?",
    criteria: {
      true:
        "The answer supplies the necessary conceptual prerequisites before relying on them.",
      false:
        "The answer relies on unexplained concepts that are part of the user's current learning goal."
    }
  },
  clarification_needed: {
    type: "noul",
    instructions:
      "Was the intended abstraction level too ambiguous to choose a reasonable answer sequence without asking a clarifying question?",
    criteria: {
      true:
        "Multiple materially different levels were equally plausible from the available context.",
      false:
        "The request and preceding context provided enough evidence to choose a reasonable level."
    }
  }
};

export const DEFAULT_THRESHOLDS = {
  clarificationNeeded: 0.7,
  abstractionMismatch: 0.7,
  prematureSpecificity: 0.65,
  progressiveDisclosureMinimum: 0.5,
  prerequisiteFitMinimum: 0.3
};

export function buildState(input) {
  if (!input || typeof input !== "object") {
    throw new TypeError("input must be an object");
  }
  if (typeof input.userRequest !== "string" || !input.userRequest.trim()) {
    throw new TypeError("userRequest must be a non-empty string");
  }
  if (
    typeof input.candidateAnswer !== "string" ||
    !input.candidateAnswer.trim()
  ) {
    throw new TypeError("candidateAnswer must be a non-empty string");
  }
  if (
    input.precedingContext !== undefined &&
    (!Array.isArray(input.precedingContext) ||
      input.precedingContext.some((value) => typeof value !== "string"))
  ) {
    throw new TypeError("precedingContext must be an array of strings");
  }

  return {
    evaluation_task:
      "Infer the requested abstraction level and evaluate whether the answer begins and progresses at an appropriate level. Do not answer the user request.",
    preceding_context: input.precedingContext ?? [],
    user_request: input.userRequest,
    candidate_answer: input.candidateAnswer
  };
}

function noul(answers, name) {
  const value = answers?.[name]?.noul;
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`Jev response is missing a valid Noul answer: ${name}`);
  }
  return value;
}

function choice(answers, name) {
  const answer = answers?.[name];
  if (!answer || typeof answer.choice !== "string") {
    throw new Error(`Jev response is missing a valid Choice answer: ${name}`);
  }
  return answer;
}

export function decideEvaluation(scores, thresholds = DEFAULT_THRESHOLDS) {
  if (scores.clarificationNeeded >= thresholds.clarificationNeeded) {
    return "ask_clarifying_question";
  }
  if (
    scores.abstractionMismatch >= thresholds.abstractionMismatch ||
    scores.prerequisiteFit < thresholds.prerequisiteFitMinimum
  ) {
    return "restructure";
  }
  if (
    scores.prematureSpecificity >= thresholds.prematureSpecificity ||
    scores.progressiveDisclosure < thresholds.progressiveDisclosureMinimum
  ) {
    return "revise_entry";
  }
  return "pass";
}

export async function evaluateAnswer(input, options = {}) {
  const apiKey = options.apiKey ?? process.env.TYPESAFE_API_KEY;
  if (!apiKey) throw new Error("TYPESAFE_API_KEY is not configured");

  const fetcher = options.fetch ?? fetch;
  const response = await fetcher(options.baseUrl ?? DEFAULT_BASE_URL, {
    method: "POST",
    headers: {
      authorization: `Bearer ${apiKey}`,
      "content-type": "application/json"
    },
    body: JSON.stringify({
      model: options.model ?? DEFAULT_MODEL,
      state: buildState(input),
      questions: QUESTIONS
    })
  });

  const body = await response.text();
  if (!response.ok) {
    throw new Error(`Jev request failed (${response.status}): ${body.slice(0, 300)}`);
  }

  let parsed;
  try {
    parsed = JSON.parse(body);
  } catch {
    throw new Error("Jev returned malformed JSON");
  }

  const requestedLevel = choice(parsed.answers, "requested_level");
  const answerEntryLevel = choice(parsed.answers, "answer_entry_level");
  const scores = {
    abstractionMismatch: noul(parsed.answers, "abstraction_mismatch"),
    prematureSpecificity: noul(parsed.answers, "premature_specificity"),
    progressiveDisclosure: noul(parsed.answers, "progressive_disclosure"),
    prerequisiteFit: noul(parsed.answers, "prerequisite_fit"),
    clarificationNeeded: noul(parsed.answers, "clarification_needed")
  };

  return {
    requestedLevel: requestedLevel.choice,
    requestedLevelConfidence: requestedLevel.confidence,
    answerEntryLevel: answerEntryLevel.choice,
    answerEntryLevelConfidence: answerEntryLevel.confidence,
    scores,
    decision: decideEvaluation(
      scores,
      options.thresholds ?? DEFAULT_THRESHOLDS
    ),
    model: parsed.model,
    usage: parsed.usage,
    rawAnswers: parsed.answers
  };
}

