/**
 * lifeScienceLabTemplates.ts — 生命科学实验室模板聚合入口
 *
 * 当前结构：
 *   - base：显微镜观察、植物细胞结构、动物细胞结构；
 *   - ext：光合作用、细胞呼吸、蒸腾作用；
 *   - human：血液循环、消化过程、人体呼吸；
 *   - genetics：有丝分裂、减数分裂、DNA与染色体、
 *     基因突变与染色体变异；
 *   - ecology：食物链与食物网、生态系统能量流动、
 *     种群数量变化、生态系统中的碳循环、
 *     生态系统中的氮循环、群落演替与生态恢复、
 *     捕食与竞争的种群关系、
 *     互利共生、偏利共生与寄生、
 *     环境容纳量与密度制约；
 *   - microbiology：细菌结构与形态、病毒侵染与复制、人体免疫防御；
 *   - regulation：反射弧与神经调节、血糖调节、体温调节；
 *   - regulationAdvanced：神经冲动的产生与传导、
 *     突触传递与兴奋抑制、甲状腺激素分级调节与负反馈、
 *     感受器与刺激强度编码、感觉形成与中枢信息加工、
 *     随意运动、反射运动与反馈校正；
 *   - molecular：DNA复制、DNA转录、mRNA翻译；
 *   - biotechnology：PCR扩增与循环过程、
 *     凝胶电泳与条带判读、基因工程基本流程；
 *   - inquiry：酶活性、渗透作用、物质跨膜运输、
 *     实验设计、对照与重复、
 *     样方法与标志重捕法种群调查、
 *     实验数据、误差、曲线与结论解释；
 *   - inheritance：孟德尔一对相对性状、孟德尔两对相对性状、
 *     遗传系谱分析、伴性遗传与性染色体传递、
 *     基因连锁、交换与重组率；
 *   - evolution：自然选择、物种形成、生物多样性与分类检索；
 *   - plantGrowth：种子萌发条件、植物向性运动、
 *     植物激素与生长调节；
 *   - plantTransport：根对水和无机盐的吸收、
 *     木质部中的水和无机盐运输、韧皮部中的有机物运输；
 *   - reproduction：花的结构、传粉与受精、果实和种子的形成、
 *     动物生命周期与变态发育；
 *   - humanReproduction：人体生殖系统与生殖细胞运输、
 *     受精、卵裂与着床、胎盘与胎儿发育；
 *   - immunityDefense：先天免疫与炎症反应、
 *     体液免疫与抗体形成、疫苗接种与免疫记忆；
 *   - excretionHomeostasis：肾单位与尿液形成、
 *     水盐平衡与抗利尿激素调节、内环境物质交换与稳态。
 *
 * 当前合计：
 *   - 72个生命科学互动模板；
 *   - 25个模板分组。
 *
 * 对外导出保持不变，LifeScienceLabModal.tsx 无需修改。
 */

import type { LifeScienceLabTemplate } from './lifeScienceLabUtils'

import {
  LIFE_SCIENCE_LAB_TEMPLATES as LIFE_SCIENCE_LAB_TEMPLATES_BASE,
} from './lifeScienceLabTemplatesBase'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EXT,
} from './lifeScienceLabTemplatesExt'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_CIRCULATION,
} from './lifeScienceLabTemplatesHumanCirculation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_DIGESTION,
} from './lifeScienceLabTemplatesHumanDigestion'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_RESPIRATION,
} from './lifeScienceLabTemplatesHumanRespiration'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MITOSIS,
} from './lifeScienceLabTemplatesGeneticsMitosis'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MEIOSIS,
} from './lifeScienceLabTemplatesGeneticsMeiosis'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_DNA_CHROMOSOME,
} from './lifeScienceLabTemplatesGeneticsDnaChromosome'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MUTATION_VARIATION,
} from './lifeScienceLabTemplatesGeneticsMutationVariation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_FOOD_WEB,
} from './lifeScienceLabTemplatesEcologyFoodWeb'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_ENERGY_FLOW,
} from './lifeScienceLabTemplatesEcologyEnergyFlow'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_POPULATION,
} from './lifeScienceLabTemplatesEcologyPopulation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_CARBON_CYCLE,
} from './lifeScienceLabTemplatesEcologyCarbonCycle'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_NITROGEN_CYCLE,
} from './lifeScienceLabTemplatesEcologyNitrogenCycle'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SUCCESSION_RESTORATION,
} from './lifeScienceLabTemplatesEcologySuccessionRestoration'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_PREDATION_COMPETITION,
} from './lifeScienceLabTemplatesEcologyPredationCompetition'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SYMBIOSIS_PARASITISM,
} from './lifeScienceLabTemplatesEcologySymbiosisParasitism'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_DENSITY_DEPENDENCE,
} from './lifeScienceLabTemplatesEcologyDensityDependence'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_BACTERIA,
} from './lifeScienceLabTemplatesMicrobiologyBacteria'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_VIRUS,
} from './lifeScienceLabTemplatesMicrobiologyVirus'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_IMMUNITY,
} from './lifeScienceLabTemplatesMicrobiologyImmunity'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_REFLEX,
} from './lifeScienceLabTemplatesRegulationReflex'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_BLOOD_GLUCOSE,
} from './lifeScienceLabTemplatesRegulationBloodGlucose'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_TEMPERATURE,
} from './lifeScienceLabTemplatesRegulationTemperature'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_ACTION_POTENTIAL,
} from './lifeScienceLabTemplatesRegulationActionPotential'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SYNAPTIC_TRANSMISSION,
} from './lifeScienceLabTemplatesRegulationSynapticTransmission'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_THYROID_AXIS,
} from './lifeScienceLabTemplatesRegulationThyroidAxis'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_RECEPTOR,
} from './lifeScienceLabTemplatesRegulationSensoryReceptor'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_PROCESSING,
} from './lifeScienceLabTemplatesRegulationSensoryProcessing'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_MOTOR_FEEDBACK,
} from './lifeScienceLabTemplatesRegulationMotorFeedback'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_DNA_REPLICATION,
} from './lifeScienceLabTemplatesMolecularDnaReplication'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSCRIPTION,
} from './lifeScienceLabTemplatesMolecularTranscription'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSLATION,
} from './lifeScienceLabTemplatesMolecularTranslation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_PCR_AMPLIFICATION,
} from './lifeScienceLabTemplatesBiotechnologyPcrAmplification'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GEL_ELECTROPHORESIS,
} from './lifeScienceLabTemplatesBiotechnologyGelElectrophoresis'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GENETIC_ENGINEERING,
} from './lifeScienceLabTemplatesBiotechnologyGeneticEngineering'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_ENZYME_ACTIVITY,
} from './lifeScienceLabTemplatesInquiryEnzymeActivity'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_OSMOSIS_CELL,
} from './lifeScienceLabTemplatesInquiryOsmosisCell'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_MEMBRANE_TRANSPORT,
} from './lifeScienceLabTemplatesInquiryMembraneTransport'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_EXPERIMENTAL_DESIGN,
} from './lifeScienceLabTemplatesInquiryExperimentalDesign'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_POPULATION_SAMPLING,
} from './lifeScienceLabTemplatesInquiryPopulationSampling'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_DATA_INTERPRETATION,
} from './lifeScienceLabTemplatesInquiryDataInterpretation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_MONOHYBRID,
} from './lifeScienceLabTemplatesInheritanceMendelMonohybrid'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_DIHYBRID,
} from './lifeScienceLabTemplatesInheritanceMendelDihybrid'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_PEDIGREE_ANALYSIS,
} from './lifeScienceLabTemplatesInheritancePedigreeAnalysis'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_SEX_LINKED,
} from './lifeScienceLabTemplatesInheritanceSexLinked'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_LINKAGE_RECOMBINATION,
} from './lifeScienceLabTemplatesInheritanceLinkageRecombination'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_NATURAL_SELECTION,
} from './lifeScienceLabTemplatesEvolutionNaturalSelection'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_SPECIATION_ISOLATION,
} from './lifeScienceLabTemplatesEvolutionSpeciationIsolation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_BIODIVERSITY_CLASSIFICATION,
} from './lifeScienceLabTemplatesEvolutionBiodiversityClassification'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_SEED_GERMINATION,
} from './lifeScienceLabTemplatesPlantSeedGermination'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_TROPISM,
} from './lifeScienceLabTemplatesPlantTropism'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_HORMONE_GROWTH,
} from './lifeScienceLabTemplatesPlantHormoneGrowth'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_ROOT_ABSORPTION,
} from './lifeScienceLabTemplatesPlantRootAbsorption'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_XYLEM_TRANSPORT,
} from './lifeScienceLabTemplatesPlantXylemTransport'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_PLANT_PHLOEM_TRANSPORT,
} from './lifeScienceLabTemplatesPlantPhloemTransport'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FLOWER,
} from './lifeScienceLabTemplatesReproductionFlower'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FRUIT_SEED,
} from './lifeScienceLabTemplatesReproductionFruitSeed'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_ANIMAL_LIFE_CYCLE,
} from './lifeScienceLabTemplatesReproductionAnimalLifeCycle'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_REPRODUCTION_SYSTEM,
} from './lifeScienceLabTemplatesHumanReproductionSystem'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_FERTILIZATION_IMPLANTATION,
} from './lifeScienceLabTemplatesHumanFertilizationImplantation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_PLACENTA_FETAL_DEVELOPMENT,
} from './lifeScienceLabTemplatesHumanPlacentaFetalDevelopment'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_INNATE_INFLAMMATION,
} from './lifeScienceLabTemplatesImmunityInnateInflammation'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_HUMORAL_ANTIBODY,
} from './lifeScienceLabTemplatesImmunityHumoralAntibody'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_VACCINATION_MEMORY,
} from './lifeScienceLabTemplatesImmunityVaccinationMemory'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_NEPHRON_URINE,
} from './lifeScienceLabTemplatesExcretionNephronUrine'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_WATER_SALT_ADH,
} from './lifeScienceLabTemplatesExcretionWaterSaltAdh'

import {
  LIFE_SCIENCE_LAB_TEMPLATES_HOMEOSTASIS_INTERNAL_ENVIRONMENT,
} from './lifeScienceLabTemplatesHomeostasisInternalEnvironment'

export const LIFE_SCIENCE_LAB_TEMPLATES: LifeScienceLabTemplate[] = [
  ...LIFE_SCIENCE_LAB_TEMPLATES_BASE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EXT,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_CIRCULATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_DIGESTION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_RESPIRATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MITOSIS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MEIOSIS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_DNA_CHROMOSOME,
  ...LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MUTATION_VARIATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_FOOD_WEB,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_ENERGY_FLOW,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_POPULATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_CARBON_CYCLE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_NITROGEN_CYCLE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SUCCESSION_RESTORATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_PREDATION_COMPETITION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SYMBIOSIS_PARASITISM,
  ...LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_DENSITY_DEPENDENCE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_BACTERIA,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_VIRUS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_IMMUNITY,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_REFLEX,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_BLOOD_GLUCOSE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_TEMPERATURE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_ACTION_POTENTIAL,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SYNAPTIC_TRANSMISSION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_THYROID_AXIS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_RECEPTOR,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_PROCESSING,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_MOTOR_FEEDBACK,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_DNA_REPLICATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSCRIPTION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSLATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_PCR_AMPLIFICATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GEL_ELECTROPHORESIS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GENETIC_ENGINEERING,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_ENZYME_ACTIVITY,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_OSMOSIS_CELL,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_MEMBRANE_TRANSPORT,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_EXPERIMENTAL_DESIGN,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_POPULATION_SAMPLING,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_DATA_INTERPRETATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_MONOHYBRID,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_DIHYBRID,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_PEDIGREE_ANALYSIS,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_SEX_LINKED,
  ...LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_LINKAGE_RECOMBINATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_NATURAL_SELECTION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_SPECIATION_ISOLATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_BIODIVERSITY_CLASSIFICATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_SEED_GERMINATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_TROPISM,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_HORMONE_GROWTH,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_ROOT_ABSORPTION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_XYLEM_TRANSPORT,
  ...LIFE_SCIENCE_LAB_TEMPLATES_PLANT_PHLOEM_TRANSPORT,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FLOWER,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FRUIT_SEED,
  ...LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_ANIMAL_LIFE_CYCLE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_REPRODUCTION_SYSTEM,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_FERTILIZATION_IMPLANTATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_PLACENTA_FETAL_DEVELOPMENT,
  ...LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_INNATE_INFLAMMATION,
  ...LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_HUMORAL_ANTIBODY,
  ...LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_VACCINATION_MEMORY,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_NEPHRON_URINE,
  ...LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_WATER_SALT_ADH,
  ...LIFE_SCIENCE_LAB_TEMPLATES_HOMEOSTASIS_INTERNAL_ENVIRONMENT,
]

export function getLifeScienceLabGroups(): {
  group: string
  items: LifeScienceLabTemplate[]
}[] {
  const groups: {
    group: string
    items: LifeScienceLabTemplate[]
  }[] = []

  for (const template of LIFE_SCIENCE_LAB_TEMPLATES) {
    let group = groups.find(
      item => item.group === template.group,
    )

    if (!group) {
      group = {
        group: template.group,
        items: [],
      }

      groups.push(group)
    }

    group.items.push(template)
  }

  return groups
}

export function findLifeScienceLabTemplate(
  id: string,
): LifeScienceLabTemplate | undefined {
  return LIFE_SCIENCE_LAB_TEMPLATES.find(
    template => template.id === id,
  )
}
