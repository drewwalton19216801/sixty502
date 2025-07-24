window.BENCHMARK_DATA = {
  "lastUpdate": 1753386592819,
  "repoUrl": "https://github.com/drewwalton19216801/sixty502",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "email": "drewwalton19216801@gmail.com",
            "name": "Drew Walton",
            "username": "drewwalton19216801"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "42abf372e74a528672b2fcea85d9a8efeb81f730",
          "message": "Merge pull request #20 from drewwalton19216801/feature/benchmark-regression-testing\n\nFix GitHub Actions benchmark workflow configuration",
          "timestamp": "2025-07-24T13:48:57-06:00",
          "tree_id": "44f373b5cc2c1e0c9cf19c1f82ed70ab74eec692",
          "url": "https://github.com/drewwalton19216801/sixty502/commit/42abf372e74a528672b2fcea85d9a8efeb81f730"
        },
        "date": 1753386592477,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkSingleInstruction/LDA_IMM",
            "value": 37.35,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31442406 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/LDA_IMM - ns/op",
            "value": 37.35,
            "unit": "ns/op",
            "extra": "31442406 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/LDA_IMM - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31442406 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/LDA_IMM - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31442406 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/STA_ABS",
            "value": 38.45,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31281205 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/STA_ABS - ns/op",
            "value": 38.45,
            "unit": "ns/op",
            "extra": "31281205 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/STA_ABS - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31281205 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/STA_ABS - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31281205 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/ADC_IMM",
            "value": 41.27,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "29028073 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/ADC_IMM - ns/op",
            "value": 41.27,
            "unit": "ns/op",
            "extra": "29028073 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/ADC_IMM - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "29028073 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/ADC_IMM - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "29028073 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JMP_ABS",
            "value": 33.1,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "36153532 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JMP_ABS - ns/op",
            "value": 33.1,
            "unit": "ns/op",
            "extra": "36153532 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JMP_ABS - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "36153532 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JMP_ABS - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "36153532 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/BNE_REL",
            "value": 27.31,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "43002138 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/BNE_REL - ns/op",
            "value": 27.31,
            "unit": "ns/op",
            "extra": "43002138 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/BNE_REL - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "43002138 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/BNE_REL - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "43002138 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JSR_RTS",
            "value": 51.15,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "23497209 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JSR_RTS - ns/op",
            "value": 51.15,
            "unit": "ns/op",
            "extra": "23497209 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JSR_RTS - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "23497209 times\n4 procs"
          },
          {
            "name": "BenchmarkSingleInstruction/JSR_RTS - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "23497209 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUClock",
            "value": 5.42,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "221828528 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUClock - ns/op",
            "value": 5.42,
            "unit": "ns/op",
            "extra": "221828528 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUClock - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "221828528 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUClock - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "221828528 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/ZeroPage",
            "value": 47.03,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "25570060 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/ZeroPage - ns/op",
            "value": 47.03,
            "unit": "ns/op",
            "extra": "25570060 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/ZeroPage - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "25570060 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/ZeroPage - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "25570060 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Absolute",
            "value": 48.32,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "24869616 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Absolute - ns/op",
            "value": 48.32,
            "unit": "ns/op",
            "extra": "24869616 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Absolute - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "24869616 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Absolute - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "24869616 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indexed",
            "value": 48.71,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "24678603 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indexed - ns/op",
            "value": 48.71,
            "unit": "ns/op",
            "extra": "24678603 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indexed - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "24678603 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indexed - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "24678603 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indirect",
            "value": 49.97,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "23880830 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indirect - ns/op",
            "value": 49.97,
            "unit": "ns/op",
            "extra": "23880830 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indirect - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "23880830 times\n4 procs"
          },
          {
            "name": "BenchmarkMemoryAccess/Indirect - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "23880830 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ADC",
            "value": 39.29,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "28479436 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ADC - ns/op",
            "value": 39.29,
            "unit": "ns/op",
            "extra": "28479436 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ADC - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "28479436 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ADC - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "28479436 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/SBC",
            "value": 39.67,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "29118105 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/SBC - ns/op",
            "value": 39.67,
            "unit": "ns/op",
            "extra": "29118105 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/SBC - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "29118105 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/SBC - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "29118105 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/CMP",
            "value": 37.95,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31164462 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/CMP - ns/op",
            "value": 37.95,
            "unit": "ns/op",
            "extra": "31164462 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/CMP - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31164462 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/CMP - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31164462 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/AND",
            "value": 37.35,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32027779 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/AND - ns/op",
            "value": 37.35,
            "unit": "ns/op",
            "extra": "32027779 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/AND - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32027779 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/AND - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32027779 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ORA",
            "value": 37.33,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32280406 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ORA - ns/op",
            "value": 37.33,
            "unit": "ns/op",
            "extra": "32280406 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ORA - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32280406 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/ORA - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32280406 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/EOR",
            "value": 37.33,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32180964 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/EOR - ns/op",
            "value": 37.33,
            "unit": "ns/op",
            "extra": "32180964 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/EOR - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32180964 times\n4 procs"
          },
          {
            "name": "BenchmarkArithmetic/EOR - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32180964 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_Taken",
            "value": 33.09,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "36368432 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_Taken - ns/op",
            "value": 33.09,
            "unit": "ns/op",
            "extra": "36368432 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_Taken - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "36368432 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_Taken - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "36368432 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_NotTaken",
            "value": 28.22,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "42536053 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_NotTaken - ns/op",
            "value": 28.22,
            "unit": "ns/op",
            "extra": "42536053 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_NotTaken - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "42536053 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BEQ_NotTaken - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "42536053 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_Taken",
            "value": 33.04,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "36237962 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_Taken - ns/op",
            "value": 33.04,
            "unit": "ns/op",
            "extra": "36237962 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_Taken - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "36237962 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_Taken - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "36237962 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_NotTaken",
            "value": 28.14,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "42458838 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_NotTaken - ns/op",
            "value": 28.14,
            "unit": "ns/op",
            "extra": "42458838 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_NotTaken - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "42458838 times\n4 procs"
          },
          {
            "name": "BenchmarkBranching/BNE_NotTaken - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "42458838 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHA",
            "value": 29.92,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "39857577 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHA - ns/op",
            "value": 29.92,
            "unit": "ns/op",
            "extra": "39857577 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHA - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "39857577 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHA - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "39857577 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLA",
            "value": 35.39,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "34011384 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLA - ns/op",
            "value": 35.39,
            "unit": "ns/op",
            "extra": "34011384 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLA - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "34011384 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLA - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "34011384 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHP",
            "value": 30,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "40106037 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHP - ns/op",
            "value": 30,
            "unit": "ns/op",
            "extra": "40106037 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHP - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "40106037 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PHP - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "40106037 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLP",
            "value": 34.5,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "34644864 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLP - ns/op",
            "value": 34.5,
            "unit": "ns/op",
            "extra": "34644864 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLP - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "34644864 times\n4 procs"
          },
          {
            "name": "BenchmarkStackOperations/PLP - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "34644864 times\n4 procs"
          },
          {
            "name": "BenchmarkCompleteProgram/CounterLoop",
            "value": 186.6,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "6422599 times\n4 procs"
          },
          {
            "name": "BenchmarkCompleteProgram/CounterLoop - ns/op",
            "value": 186.6,
            "unit": "ns/op",
            "extra": "6422599 times\n4 procs"
          },
          {
            "name": "BenchmarkCompleteProgram/CounterLoop - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "6422599 times\n4 procs"
          },
          {
            "name": "BenchmarkCompleteProgram/CounterLoop - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "6422599 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/IRQ",
            "value": 13.58,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "88340019 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/IRQ - ns/op",
            "value": 13.58,
            "unit": "ns/op",
            "extra": "88340019 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/IRQ - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "88340019 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/IRQ - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "88340019 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/NMI",
            "value": 13.69,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "87977344 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/NMI - ns/op",
            "value": 13.69,
            "unit": "ns/op",
            "extra": "87977344 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/NMI - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "87977344 times\n4 procs"
          },
          {
            "name": "BenchmarkInterrupts/NMI - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "87977344 times\n4 procs"
          }
        ]
      }
    ]
  }
}